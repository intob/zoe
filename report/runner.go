package report

import (
	"context"
	"fmt"
	"time"

	"github.com/swissinfo-ch/zoe/ev"
	"github.com/swissinfo-ch/zoe/worker"
)

type RunnerCfg struct {
	Filename          string
	BlockSize         int
	WorkerPoolSize    int
	MinReportInterval time.Duration
	Jobs              map[string]*Job
}

type Runner struct {
	filename                string
	blockSize               int
	workerPoolSize          int
	minReportInterval       time.Duration
	jobs                    map[string]*Job
	results                 map[string]*Result
	jobDone                 chan *JobDone
	events                  chan *ev.Ev
	fileSize                int64
	currentReportEventCount uint32
	lastReportEventCount    uint32
	lastReportDuration      time.Duration
	lastReportTime          time.Time
}

type Job struct {
	Report Report
	events chan *ev.Ev // events will be sent to this channel, and closed when the job is done
}

type JobDone struct {
	Name   string
	Result *Result
}

// NewRunner creates & starts a new report runner
func NewRunner(cfg *RunnerCfg) *Runner {
	r := &Runner{
		filename:          cfg.Filename,
		blockSize:         cfg.BlockSize,
		workerPoolSize:    cfg.WorkerPoolSize,
		minReportInterval: cfg.MinReportInterval,
		jobs:              cfg.Jobs,
		results:           make(map[string]*Result, len(cfg.Jobs)),
	}
	// Start the report runner
	go func() {
		for {
			tStart := time.Now()
			// TODO add context
			r.run(context.TODO())
			r.lastReportDuration = time.Since(tStart)
			r.lastReportTime = time.Now()
			evPerSec := FmtCount(uint32(float64(r.lastReportEventCount) / r.lastReportDuration.Seconds()))
			fmt.Printf("\r%s // %s // reporting took %v for %s evs at %s ev/s",
				r.lastReportTime.Format(time.RFC3339),
				FmtFileSize(r.fileSize),
				r.lastReportDuration,
				FmtCount(r.lastReportEventCount),
				evPerSec)
			fmt.Print("\033[0K") // flush line
			// limit report running rate
			if r.lastReportDuration < r.minReportInterval {
				time.Sleep(r.minReportInterval - r.lastReportDuration)
			}
		}
	}()
	return r
}

// Jobs returns the jobs
func (r *Runner) Jobs() map[string]*Job {
	return r.jobs
}

// Results returns the results of a job
func (r *Runner) Result(jobName string) (*Result, bool) {
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

// run generates a report for each job
func (r *Runner) run(ctx context.Context) {
	r.jobDone = make(chan *JobDone, len(r.jobs))
	// TUNING: 2024-02-09
	// buffer size equal to block size is optimal
	r.events = make(chan *ev.Ev, r.blockSize)
	for jobName, job := range r.jobs {
		// TUNING: 2024-02-10
		// job events chan buffer size 2 seems optimal,
		// otherwise sendEventsCollectResults will block
		job.events = make(chan *ev.Ev, 1)
		go r.generateJobReport(job, jobName)
	}
	go r.readEventsFromFile()
	r.sendEventsCollectResults(ctx)
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

func (r *Runner) sendEventsCollectResults(ctx context.Context) {
	workerPool := worker.NewPool(r.workerPoolSize)
	workerPool.Start()

	runningJobs := make(map[string]*Job, len(r.jobs))

	// Copy job references to manage individual job event channels safely
	for name, job := range r.jobs {
		runningJobs[name] = job
	}

	r.currentReportEventCount = 0

loop:
	for {
		select {
		case e, ok := <-r.events:
			if !ok {
				// If the events channel is closed, it's time to cleanup and exit
				break loop
			}

			// Dispatch a job to handle the event
			workerPool.Dispatch(func() {
				for _, job := range runningJobs {
					// Before attempting to send, check if context has been cancelled
					if ctx.Err() != nil {
						return // Avoid sending on closed channel if shutdown is initiated
					}
					select {
					case job.events <- e:
						// Event sent successfully
					case <-time.After(time.Millisecond * 100):
						// Optionally handle the timeout case here
					case <-ctx.Done():
						// Shutdown signal received, exit the dispatched job
						return
					}
				}
			})

			// Increment the event count
			r.currentReportEventCount++
		case <-ctx.Done():
			// Shutdown signal received, exit the loop
			break loop
		}
	}

	// Wait for all dispatched jobs to finish
	workerPool.StopAndWait()

	// Safely close all job event channels after all work is done
	for _, job := range runningJobs {
		close(job.events)
	}

	// Collect results
	done := 0
	for {
		j, ok := <-r.jobDone
		if !ok {
			fmt.Println("jobDone channel closed unexpectedly")
			break
		}
		r.results[j.Name] = j.Result
		done++
		if done >= len(r.jobs) {
			break
		}
	}

	r.lastReportEventCount = r.currentReportEventCount
}
