package report

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/swissinfo-ch/lstn/ev"
	"google.golang.org/protobuf/proto"
)

type Runner struct {
	Results  map[string][]byte
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
	err := r.readEvents()
	if err != nil {
		panic(fmt.Sprintf("failed to read events: %v", err))
	}
}

func (r *Runner) readEvents() error {
	file, err := os.Open(r.filename)
	if err != nil {
		return fmt.Errorf("failed to open file for reading: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		// Read the length as a single byte
		lengthByte, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break // End of file reached, stop reading
			}
			return fmt.Errorf("failed to read event length: %w", err)
		}

		// Convert the length byte to an integer
		length := int(lengthByte)

		// Allocate a slice for the data of the event
		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			return fmt.Errorf("failed to read event payload: %w", err)
		}

		// Unmarshal the protobuf event
		e := &ev.Ev{}
		if err := proto.Unmarshal(data, e); err != nil {
			return fmt.Errorf("failed to unmarshal protobuf: %w", err)
		}

		// Send the event to all jobs
		for _, job := range r.jobs {
			job.events <- e
		}
	}

	// Close all job event channels after reading all events
	for _, job := range r.jobs {
		close(job.events)
	}

	return nil
}
