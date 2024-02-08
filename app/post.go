package app

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/swissinfo-ch/lstn/ev"
	"google.golang.org/protobuf/proto"
)

// Handle event input
func (a *App) handlePost(w http.ResponseWriter, r *http.Request) {
	evType, ok := ev.EvType_value[r.Header.Get("TYPE")]
	if !ok {
		http.Error(w, "invalid header TYPE, must be one of LOAD, UNLOAD or TIME", http.StatusBadRequest)
		return
	}
	usr, err := strconv.ParseUint(r.Header.Get("USR"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("err to parse uint32 in header USR: %w", err).Error(), http.StatusBadRequest)
		return
	}
	sess, err := strconv.ParseUint(r.Header.Get("SESS"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("err to parse uint32 in header SESS: %w", err).Error(), http.StatusBadRequest)
		return
	}
	cid, err := strconv.ParseUint(r.Header.Get("CID"), 10, 32)
	if err != nil {
		http.Error(w, fmt.Errorf("err to parse uint32 in header CID: %w", err).Error(), http.StatusBadRequest)
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
		scrolled, err := strconv.ParseFloat(r.Header.Get("SCROLLED"), 32)
		if err != nil {
			http.Error(w, fmt.Errorf("failed to parse SCROLLED: %w", err).Error(), http.StatusBadRequest)
			return
		}
		scrolled32 := float32(scrolled)
		e.Scrolled = &scrolled32
	case ev.EvType_TIME:
		pageSeconds, err := strconv.ParseUint(r.Header.Get("PAGE_SECONDS"), 10, 32)
		if err != nil {
			http.Error(w, fmt.Errorf("failed to parse PAGE_SECONDS: %w", err).Error(), http.StatusBadRequest)
			return
		}
		pageSeconds32 := uint32(pageSeconds)
		e.PageSeconds = &pageSeconds32
	}
	a.events <- e
}

// writeEvents writes to the file in a loop
// TODO: add a buffer to write gzipped groups
func (a *App) writeEvents() {
	file, err := os.OpenFile(a.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Sprintf("failed to open file: %v", err))
	}
	defer file.Close()
	for {
		select {
		case e := <-a.events:
			if err := a.writeEvent(file, e); err != nil {
				panic(fmt.Sprintf("failed to write event: %v", err))
			}
		case <-a.ctx.Done():
			return
		}
	}
}

// writeEvent writes the event to the file
// TODO: reverse the order of the parts
// to optimize reading most recent events
func (a *App) writeEvent(w io.Writer, e *ev.Ev) error {
	// Marshal the protobuf event.
	data, err := proto.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}

	// Check if data length exceeds the maximum size of 35 bytes.
	// TODO: calculate 35 from the protobuf definition.
	if len(data) > 35 {
		return fmt.Errorf("event size %d exceeds maximum of 35 bytes", len(data))
	}

	// Prepend the size as a single byte.
	// Since we know the maximum size is 35 bytes, a single byte for size is sufficient.
	sizeByte := byte(len(data))         // Convert the length of data to a single byte.
	buf := make([]byte, 0, 1+len(data)) // Allocate buffer for size byte and data.
	buf = append(buf, data...)          // Append the actual event data.
	buf = append(buf, sizeByte)         // Append size byte.

	// Write the buffer to the io.Writer.
	n, err := w.Write(buf)
	if err != nil {
		return fmt.Errorf("failed to write to buffer: %w", err)
	}
	if n != len(buf) {
		return errors.New("failed to write all bytes to buffer")
	}

	return nil
}
