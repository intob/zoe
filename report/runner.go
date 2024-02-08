package report

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/swissinfo-ch/lstn/ev"
	"google.golang.org/protobuf/proto"
)

type RunnerCfg struct {
	Filename          string
	Jobs              map[string]*Job
	MinReportInterval time.Duration
}

type Runner struct {
	results                 map[string][]byte
	jobs                    map[string]*Job
	jobDone                 chan *JobDone
	events                  chan *ev.Ev
	filename                string
	fileSize                int64
	currentReportEventCount uint32
	lastReportEventCount    uint32
	lastReportDuration      time.Duration
	lastReportTime          time.Time
	minReportInterval       time.Duration
}

type Job struct {
	Report Report
	events chan *ev.Ev // events will be sent to this channel, and closed when the job is done
}

type JobDone struct {
	Name   string
	Result []byte
}

// NewRunner creates & starts a new report runner
func NewRunner(cfg *RunnerCfg) *Runner {
	r := &Runner{
		results:           make(map[string][]byte),
		filename:          cfg.Filename,
		jobs:              cfg.Jobs,
		minReportInterval: cfg.MinReportInterval,
	}
	// Start the report runner
	go func() {
		for {
			tStart := time.Now()
			r.Run()
			r.lastReportDuration = time.Since(tStart)
			r.lastReportTime = time.Now()
			fmt.Printf("\r%s reporting took %v for %d events",
				r.lastReportTime.Format(time.RFC3339),
				r.lastReportDuration,
				r.currentReportEventCount)
			// limit report running rate
			if r.lastReportDuration < r.minReportInterval {
				time.Sleep(r.minReportInterval - r.lastReportDuration)
			}
		}
	}()
	return r
}

// Run generates a report for each job
func (r *Runner) Run() {
	r.jobDone = make(chan *JobDone)
	r.events = make(chan *ev.Ev, 100)
	for jobName, job := range r.jobs {
		job.events = make(chan *ev.Ev, 4)
		go r.generateJobReport(job, jobName)
	}
	go r.readEventsFromFile()
	r.sendEventsCollectResults()
}

// Jobs returns the jobs
func (r *Runner) Jobs() map[string]*Job {
	return r.jobs
}

// Results returns the results of a job
func (r *Runner) Results(jobName string) ([]byte, bool) {
	result, exists := r.results[jobName]
	return result, exists
}

// CurrentReportEventCount returns the number of events read for the current report
func (r *Runner) CurrentReportEventCount() uint32 {
	return r.currentReportEventCount
}

// LastReportEventCount returns the number of events read for the last report
func (r *Runner) LastReportEventCount() uint32 {
	return r.lastReportEventCount
}

// LastReportDuration returns the duration of the last report
func (r *Runner) LastReportDuration() time.Duration {
	return r.lastReportDuration
}

// LastReportTime returns the time of the last report
func (r *Runner) LastReportTime() time.Time {
	return r.lastReportTime
}

// FileSize returns the size of the file
func (r *Runner) FileSize() int64 {
	return r.fileSize
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

// sendEventsCollectResults sends events to the jobs and collects the results
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
				for name, job := range runningJobs {
					close(job.events)
					delete(runningJobs, name)
				}
				continue
			}
			for _, job := range runningJobs {
				select {
				case job.events <- e:
					// event sent
				case <-time.After(time.Microsecond * 50):
					// timeout
				}
			}
		case j := <-r.jobDone:
			r.results[j.Name] = j.Result
			delete(runningJobs, j.Name)
			countDone++
			if len(r.jobs) == countDone {
				return
			}
		}
	}
}

// readEventsFromFile reads events from a file and sends them to the events channel
func (r *Runner) readEventsFromFile() {
	// Reset the event count
	r.currentReportEventCount = 0

	// Open the file
	file, err := os.Open(r.filename)
	if err != nil {
		panic(fmt.Sprintf("failed to open file for reading: %v", err))
	}
	defer file.Close()

	// Get the file size
	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}
	r.fileSize = fileInfo.Size()

	// Starting from the end of the file
	offset := r.fileSize

	for offset > 0 {
		// Move back to read the length
		offset -= 1
		_, err := file.Seek(offset, io.SeekStart)
		if err != nil {
			panic(fmt.Sprintf("failed to seek in file: %v", err))
		}

		// Read the event length
		lengthByte := make([]byte, 1)
		_, err = file.Read(lengthByte)
		if err != nil {
			panic(fmt.Sprintf("failed to read event length: %v", err))
		}
		length := int(lengthByte[0])

		// Move back to read the event payload
		offset -= int64(length)
		_, err = file.Seek(offset, io.SeekStart)
		if err != nil {
			panic(fmt.Sprintf("failed to seek in file: %v", err))
		}

		// Read the event payload
		data := make([]byte, length)
		_, err = file.Read(data)
		if err != nil {
			panic(fmt.Sprintf("failed to read event payload: %v", err))
		}

		// Unmarshal the protobuf event
		e := &ev.Ev{}
		if err := proto.Unmarshal(data, e); err != nil {
			panic(fmt.Sprintf("failed to unmarshal protobuf: %v", err))
		}

		// Send the event to the channel
		r.events <- e

		// Increment the event count
		r.currentReportEventCount++
	}
	// Close the events channel
	close(r.events)
	// Update the last report event count
	r.lastReportEventCount = r.currentReportEventCount
}
