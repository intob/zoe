package report

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/swissinfo-ch/lstn/ev"
	"google.golang.org/protobuf/proto"
)

type RunnerCfg struct {
	Filename string
	Jobs     map[string]*Job
}

type Runner struct {
	results  map[string][]byte
	jobs     map[string]*Job
	jobDone  chan *JobDone
	events   chan *ev.Ev
	filename string
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
		job.events = make(chan *ev.Ev, 2)
		go r.generateJobReport(job, jobName)
	}
	go r.readEventsFromFile()
	r.sendEventsCollectResults()
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

// dispatchEventsToJobs sends the events to the jobs
func (r *Runner) sendEventsCollectResults() {
	countDone := 0
	runningJobs := make(map[string]*Job, len(r.jobs))
	for name, job := range r.jobs {
		runningJobs[name] = job
	}
	for {
		select {
		case e, ok := <-r.events:
			if !ok {
				return
			}
			for jobName, job := range runningJobs {
				select {
				case job.events <- e:
				case <-time.After(time.Microsecond * 50):
					fmt.Println("timeout, dropping event for job", jobName)
				}
			}
		case j := <-r.jobDone:
			r.results[j.Name] = j.Result
			fmt.Println("job done:", j.Name)
			close(runningJobs[j.Name].events)
			delete(runningJobs, j.Name)
			fmt.Println(runningJobs)
			countDone++
			if len(r.jobs) == countDone {
				return
			}
		}
	}
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
		// Read the length as a byte
		lengthByte, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break // End of file reached, stop reading
			}
			panic(fmt.Sprintf("failed to read event length: %v", err))
		}
		length := int(lengthByte)

		// read payload
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
