package app

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/swissinfo-ch/zoe/ev"
	"google.golang.org/protobuf/proto"
)

// handlePost is the HTTP handler for the POST / endpoint.
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
	block := &ev.Block{
		Evs: make([]*ev.Ev, 0, a.blockSize),
	}
	for {
		select {
		case e := <-a.events:
			block.Evs = append(block.Evs, e)
			if len(block.Evs) >= a.blockSize {
				err := a.writeBlock(block, file)
				if err != nil {
					panic(fmt.Sprintf("failed to write block: %v", err))
				}
				block.Reset()
			}
		case <-a.ctx.Done():
			return
		}
	}
}

// writeBlock gzips & writes a block to the io.Writer
func (a *App) writeBlock(block *ev.Block, w io.Writer) error {
	gzbuf := &bytes.Buffer{}
	gw := gzip.NewWriter(gzbuf)

	// Marshal the block into protobuf data
	data, err := proto.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	// Write marshaled data to gzip writer
	if _, err := gw.Write(data); err != nil {
		return fmt.Errorf("failed to write block: %w", err)
	}

	// It's crucial to close the gzip writer before accessing the buffer
	// to ensure all data is flushed and compressed
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Write gzipped block to the io.Writer
	gzippedData := gzbuf.Bytes() // Get the compressed data
	if _, err := w.Write(gzippedData); err != nil {
		return fmt.Errorf("failed to write gzipped block: %w", err)
	}

	// Convert the length of gzipped data to a 4-byte slice
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(len(gzippedData)))

	// Write the length bytes to the io.Writer
	if _, err := w.Write(lengthBytes); err != nil {
		return fmt.Errorf("failed to write gzipped block length: %w", err)
	}

	return nil
}
