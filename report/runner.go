package report

import (
	"bufio"
	"encoding/binary"
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
	r.readEvents()
}

func (r *Runner) readEvents() {
	file, err := os.Open(r.filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		length, err := binary.ReadUvarint(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			panic(err)
		}
		e := &ev.Ev{}
		if err := proto.Unmarshal(data, e); err != nil {
			panic(err)
		}
		for _, job := range r.jobs {
			job.events <- e
		}
	}
	for _, job := range r.jobs {
		close(job.events)
	}
}
