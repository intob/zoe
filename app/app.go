package app

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
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

type App struct {
	clients  map[string]*client // writer:addr or reader:addr
	clientMu sync.Mutex
	events   chan *ev.Ev
	filename string
}

type AppCfg struct {
	Filename string
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewApp(cfg *AppCfg) *App {
	a := &App{
		filename: cfg.Filename,
		clients:  make(map[string]*client),
		clientMu: sync.Mutex{},
		events:   make(chan *ev.Ev),
	}
	go a.cleanupVisitors()
	go a.writeEventsToFile()
	return a
}

func (a *App) Start() {
	mux := http.NewServeMux()
	mux.Handle("/", a.rateLimitMiddleware(
		a.corsMiddleware(
			http.HandlerFunc(a.handleRequest))))
	fmt.Println("app listening http on :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}

func (a *App) handleRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		switch r.URL.Path {
		case "/data":
			a.handleGetData(w, r)
		case "/js":
			a.handleGetJS(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	case "POST":
		a.handlePost(w, r)
	}
}

// Handle event input
func (a *App) handlePost(w http.ResponseWriter, r *http.Request) {
	evType, ok := ev.EvType_value[r.Header.Get("X_TYPE")]
	if !ok {
		http.Error(w, "invalid header X_TYPE, must be one of LOAD, UNLOAD or TIME", http.StatusBadRequest)
		return
	}
	usr, err := strconv.ParseUint(r.Header.Get("X_USR"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("err to parse uint32 in header X_USR: %w", err).Error(), http.StatusBadRequest)
		return
	}
	sess, err := strconv.ParseUint(r.Header.Get("X_SESS"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("err to parse uint32 in header X_SESS: %w", err).Error(), http.StatusBadRequest)
		return
	}
	cid, err := strconv.ParseUint(r.Header.Get("X_CID"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("err to parse uint32 in header X_CID: %w", err).Error(), http.StatusBadRequest)
		return
	}
	e := &ev.Ev{
		Time:   uint32(time.Now().Unix()),
		EvType: ev.EvType(evType),
		Usr:    uint32(usr),
		Sess:   uint32(sess),
		Cid:    uint32(cid),
	}
	switch e.EvType {
	case ev.EvType_UNLOAD:
		scrolled, err := strconv.ParseFloat(r.Header.Get("X_SCROLLED"), 32)
		if err != nil {
			http.Error(w, fmt.Errorf("failed to parse X_SCROLLED: %w", err).Error(), http.StatusBadRequest)
			return
		}
		scrolled32 := float32(scrolled)
		e.Scrolled = &scrolled32
	case ev.EvType_TIME:
		pageSeconds, err := strconv.ParseUint(r.Header.Get("X_PAGE_SECONDS"), 10, 32)
		if err != nil {
			http.Error(w, fmt.Errorf("failed to parse X_PAGE_SECONDS: %w", err).Error(), http.StatusBadRequest)
			return
		}
		pageSeconds32 := uint32(pageSeconds)
		e.PageSeconds = &pageSeconds32
	}
	a.events <- e
}

// Write all data to response writer
func (a *App) handleGetData(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open(a.filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	_, err = io.Copy(gz, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *App) handleGetJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=3600, must-revalidate")
	w.Header().Set("Content-Type", "text/javascript")
	http.ServeFile(w, r, "client.js")
}

// Write to file what is sent on events chan
func (a *App) writeEventsToFile() {
	file, err := os.OpenFile(a.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	defer file.Close()
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case e := <-a.events:
			if err := a.writeEvent(writer, e); err != nil {
				fmt.Println("failed to write event:", err)
			}
		case <-ticker.C:
			writer.Flush()
		}
	}
}

func (a *App) writeEvent(w *bufio.Writer, e *ev.Ev) error {
	data, err := proto.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}
	sizeBuf := make([]byte, binary.MaxVarintLen64)
	sizeSize := binary.PutUvarint(sizeBuf, uint64(len(data)))
	buf := make([]byte, 0, sizeSize+len(data))
	buf = append(buf, sizeBuf[:sizeSize]...)
	buf = append(buf, data...)
	_, err = w.Write(buf)
	if err != nil {
		return fmt.Errorf("failed to write to buffer: %w", err)
	}
	return nil
}

func (a *App) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := a.getRateLimiter(r)
		if !limiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: check Origin header
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "X_TYPE,X_USR,X_SESS,X_CID,X_SCROLLED,X_PAGE_SECONDS")
		next.ServeHTTP(w, r)
	})
}

func (a *App) getRateLimiter(r *http.Request) *rate.Limiter {
	addr := r.Header.Get("Fly-Client-IP")
	if addr == "" {
		addr = r.RemoteAddr // fallback when local
	}
	a.clientMu.Lock()
	defer a.clientMu.Unlock()
	key := r.Method + addr
	v, exists := a.clients[key]
	if !exists {
		var limiter *rate.Limiter
		if r.Method == "POST" { // fast rate for writing
			limiter = rate.NewLimiter(rate.Every(time.Second), 4)
		} else { // lower rate for download
			limiter = rate.NewLimiter(rate.Every(time.Second*10), 2)
		}
		a.clients[key] = &client{limiter, time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (a *App) cleanupVisitors() {
	for {
		time.Sleep(10 * time.Second)
		a.clientMu.Lock()
		for key, client := range a.clients {
			if time.Since(client.lastSeen) > 10*time.Second {
				delete(a.clients, key)
			}
		}
		a.clientMu.Unlock()
	}
}
