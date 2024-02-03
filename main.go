package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/swissinfo-ch/lstn/ev"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

type httpSvc struct {
	visitors map[string]*visitor // key is ip addr
	mu       sync.Mutex
	events   chan *ev.Ev
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func main() {
	hs := &httpSvc{
		visitors: make(map[string]*visitor),
		mu:       sync.Mutex{},
		events:   make(chan *ev.Ev),
	}
	go hs.cleanupVisitors()
	go hs.writeEventsToFile("events")
	mux := http.NewServeMux()
	mux.Handle("/", hs.rateLimitMiddleware(http.HandlerFunc(hs.handleRequest)))
	fmt.Println("listening http on :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}

func (hs *httpSvc) handleRequest(w http.ResponseWriter, r *http.Request) {
	t, ok := ev.Type_value[r.Header.Get("LSTN_T")]
	if !ok {
		http.Error(w, "header LSTN_T is invalid", http.StatusBadRequest)
		return
	}
	usr, err := strconv.ParseUint(r.Header.Get("LSTN_USR"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("failed to parse LSTN_USR: %w", err).Error(), http.StatusBadRequest)
		return
	}
	sess, err := strconv.ParseUint(r.Header.Get("LSTN_SESS"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("failed to parse LSTN_SESS: %w", err).Error(), http.StatusBadRequest)
		return
	}
	cid, err := strconv.ParseUint(r.Header.Get("LSTN_CID"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("failed to parse LSTN_CID: %w", err).Error(), http.StatusBadRequest)
		return
	}
	e := &ev.Ev{
		Time: uint32(time.Now().Unix()),
		Type: ev.Type(t),
		Usr:  uint32(usr),
		Sess: uint32(sess),
		Cid:  uint32(cid),
	}
	switch e.Type {
	case ev.Type_UNLOAD:
		scrolled, err := strconv.ParseFloat(r.Header.Get("LSTN_SCROLLED"), 32)
		if err != nil {
			http.Error(w, fmt.Errorf("failed to parse LSTN_SCROLLED: %w", err).Error(), http.StatusBadRequest)
			return
		}
		scrolled32 := float32(scrolled)
		e.Scrolled = &scrolled32
	case ev.Type_TIME:
		pageSeconds, err := strconv.ParseUint(r.Header.Get("LSTN_PAGE_SECONDS"), 10, 32)
		if err != nil {
			http.Error(w, fmt.Errorf("failed to parse LSTN_PAGE_SECONDS: %w", err).Error(), http.StatusBadRequest)
			return
		}
		pageSeconds32 := uint32(pageSeconds)
		e.PageSeconds = &pageSeconds32
	}
	hs.events <- e
}

func (hs *httpSvc) writeEventsToFile(filename string) {
	// open buffered writer
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	defer file.Close()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			<-ticker.C
			if err := writer.Flush(); err != nil {
				fmt.Println("failed to flush buffer:", err)
			}
		}
	}()

	for e := range hs.events {
		data, err := proto.Marshal(e)
		if err != nil {
			fmt.Println("failed to marshal protobuf:", err)
			continue
		}
		sizeBuf := make([]byte, binary.MaxVarintLen64)
		sizeSize := binary.PutUvarint(sizeBuf, uint64(len(data)))
		buf := make([]byte, 0, sizeSize+len(data))
		buf = append(buf, sizeBuf[:sizeSize]...)
		buf = append(buf, data...)
		_, err = writer.Write(buf)
		if err != nil {
			fmt.Println("failed to write to buffer:", err)
			continue
		}
	}
}

func (hs *httpSvc) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := hs.getVisitor(r.RemoteAddr)
		if !limiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (hs *httpSvc) getVisitor(addr string) *rate.Limiter {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	v, exists := hs.visitors[addr]
	if !exists {
		limiter := rate.NewLimiter(rate.Every(time.Second), 4)
		hs.visitors[addr] = &visitor{limiter, time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (hs *httpSvc) cleanupVisitors() {
	for {
		time.Sleep(10 * time.Second)
		hs.mu.Lock()
		for addr, v := range hs.visitors {
			if time.Since(v.lastSeen) > 10*time.Second {
				delete(hs.visitors, addr)
			}
		}
		hs.mu.Unlock()
	}
}
