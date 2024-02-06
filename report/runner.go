package report

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/swissinfo-ch/lstn/ev"
	"google.golang.org/protobuf/proto"
)

type Runner struct {
	Results  map[string][]byte
	Kill     chan struct{} // Will send a signal to kill the app
	Killed   chan struct{} // Will be closed when the app is killed
	filename string
	jobs     []*Job
}

type RunnerCfg struct {
	Filename string
	Jobs     []*Job
}

type Job struct {
	Report Report
	Name   string
	events chan *ev.Ev
}

func NewRunner(cfg *RunnerCfg) *Runner {
	return &Runner{
		Results:  make(map[string][]byte),
		Kill:     make(chan struct{}),
		Killed:   make(chan struct{}),
		filename: cfg.Filename,
		jobs:     cfg.Jobs,
	}
}

func (r *Runner) Run() {
	for _, job := range r.jobs {
		job.events = make(chan *ev.Ev)
		go func(job *Job) {
			report, err := job.Report.Generate(job.events)
			if err != nil {
				panic(err)
			}
			r.Results[job.Name] = report
		}(job)
	}
	lastGoodPos, err := r.readEvents()
	if err != nil {
		err = r.truncateFile(lastGoodPos)
		if err != nil {
			fmt.Println("failed to truncate file:", err)
		}
		os.Exit(1)
	}
}

func (r *Runner) readEvents() (int64, error) {
	file, err := os.Open(r.filename)
	if err != nil {
		panic(fmt.Sprintf("failed to open file for reading: %v", err))
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	lastGoodPos := int64(0)
	for {
		currentPos, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			panic(fmt.Sprintf("failed to seek: %v", err))
		}
		length, err := binary.ReadUvarint(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("failed to read ev length, will truncate broken data")
			return lastGoodPos, fmt.Errorf("failed to read ev length: %w", err)
		}
		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			return lastGoodPos, fmt.Errorf("failed to read ev payload: %w", err)
		}
		e := &ev.Ev{}
		if err := proto.Unmarshal(data, e); err != nil {
			return lastGoodPos, fmt.Errorf("failed to unmarshal protobuf: %w", err)
		}
		for _, job := range r.jobs {
			job.events <- e
		}
		lastGoodPos = currentPos
	}
	for _, job := range r.jobs {
		close(job.events)
	}
	return lastGoodPos, nil
}

func (r *Runner) truncateFile(lastGoodPos int64) error {
	fmt.Println("waiting for app to shutdown...")
	r.Kill <- struct{}{}
	<-r.Killed
	fmt.Println("app shutdown complete")
	file, err := os.OpenFile(r.filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for truncation: %w", err)
	}
	defer file.Close()
	if err := file.Truncate(lastGoodPos); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}
	fmt.Println("truncated file to last good position")
	return nil
}
