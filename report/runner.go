package report

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/swissinfo-ch/lstn/ev"
	"google.golang.org/protobuf/proto"
)

type RunnerCfg struct {
	Filename string
	Jobs     map[string]*Job
}

type Runner struct {
	rwMu     sync.RWMutex
	results  map[string][]byte
	jobs     map[string]*Job
	jobDone  chan *JobDone
	filename string
	events   chan *ev.Ev
}

type Job struct {
	Report Report
	events chan *ev.Ev // events will be sent to this channel, and closed when the job is done
}

type JobDone struct {
	Name   string
	Result []byte
}

// NewRunner creates a new report runner
func NewRunner(cfg *RunnerCfg) *Runner {
	return &Runner{
		results:  make(map[string][]byte),
		filename: cfg.Filename,
		jobs:     cfg.Jobs,
	}
}

// Run generates a report for each job
func (r *Runner) Run() {
	r.jobDone = make(chan *JobDone)
	r.events = make(chan *ev.Ev)
	for jobName, job := range r.jobs {
		job.events = make(chan *ev.Ev, 10)
		go r.generateJobReport(job, jobName)
	}
	go r.dispatchEventsToJobs()
	go r.readEventsFromFile()
	r.collectJobResults()
}

// Results returns the results of a job
func (r *Runner) Results(jobName string) ([]byte, bool) {
	result, exists := r.results[jobName]
	return result, exists
}

// generateReport generates a report for a job
func (r *Runner) generateJobReport(job *Job, jobName string) {
	report, err := job.Report.Generate(job.events)
	if err != nil {
		panic(err)
	}
	r.jobDone <- &JobDone{
		Name:   jobName,
		Result: report,
	}
}

// collectResults collects the results of the jobs
func (r *Runner) collectJobResults() {
	countDone := 0
	for countDone < len(r.jobs) {
		done := <-r.jobDone
		countDone++
		fmt.Println("job done:", done.Name)
		r.rwMu.Lock()
		r.results[done.Name] = done.Result
		r.jobs[done.Name].events = nil
		r.rwMu.Unlock()
	}
	fmt.Println("all jobs done")
}

// dispatchEventsToJobs sends the events to the jobs
func (r *Runner) dispatchEventsToJobs() {
	for e := range r.events {
		fmt.Println("dispatching event:", e)
		r.rwMu.RLock()
		for _, job := range r.jobs {
			if job.events == nil {
				continue
			}
			job.events <- e
			fmt.Printf("job<-e\n")
		}
		r.rwMu.RUnlock()
	}
	r.rwMu.RLock()
	for jobName, job := range r.jobs {
		if job.events == nil {
			fmt.Println("skipped closing job events channel:", jobName)
		} else {
			close(job.events)
		}
		fmt.Println("closed job events channel:", jobName)
	}
	r.rwMu.RUnlock()
}

// readEventsFromFile reads events from a file and sends them to the events channel
func (r *Runner) readEventsFromFile() {
	file, err := os.Open(r.filename)
	if err != nil {
		panic(fmt.Sprintf("failed to open file for reading: %v", err))
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	count := 0
	for {
		// Read the length as a single byte
		lengthByte, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break // End of file reached, stop reading
			}
			panic(fmt.Sprintf("failed to read event length: %v", err))
		}

		// Convert the length byte to an integer
		length := int(lengthByte)

		// Allocate a slice for the data of the event
		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			panic(fmt.Sprintf("failed to read event payload: %v", err))
		}

		// Unmarshal the protobuf event
		e := &ev.Ev{}
		if err := proto.Unmarshal(data, e); err != nil {
			panic(fmt.Sprintf("failed to unmarshal protobuf: %v", err))
		}

		r.events <- e

		count++
	}

	close(r.events)

	fmt.Printf("read %d events\n", count)
}
