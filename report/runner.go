package report

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/swissinfo-ch/zoe/ev"
	"github.com/swissinfo-ch/zoe/worker"
	"google.golang.org/protobuf/proto"
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
	results                 map[string][]byte
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
	Result []byte
}

// NewRunner creates & starts a new report runner
func NewRunner(cfg *RunnerCfg) *Runner {
	r := &Runner{
		filename:          cfg.Filename,
		blockSize:         cfg.BlockSize,
		workerPoolSize:    cfg.WorkerPoolSize,
		minReportInterval: cfg.MinReportInterval,
		jobs:              cfg.Jobs,
		results:           make(map[string][]byte),
	}
	// Start the report runner
	go func() {
		for {
			tStart := time.Now()
			r.run()
			r.lastReportDuration = time.Since(tStart)
			r.lastReportTime = time.Now()
			evPerSec := FmtCount(uint32(float64(r.currentReportEventCount) / r.lastReportDuration.Seconds()))
			fmt.Printf("\r%s reporting took %v for %s evs at %s ev/s",
				r.lastReportTime.Format(time.RFC3339),
				r.lastReportDuration,
				FmtCount(r.currentReportEventCount),
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

// run generates a report for each job
func (r *Runner) run() {
	r.jobDone = make(chan *JobDone)
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
	r.sendEventsCollectResults(context.TODO())
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

	countDone := 0
	runningJobs := make(map[string]*Job, len(r.jobs))

	// Copy job references to manage individual job event channels safely
	for name, job := range r.jobs {
		runningJobs[name] = job
	}

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
		case j := <-r.jobDone:
			r.results[j.Name] = j.Result
			delete(runningJobs, j.Name)
			countDone++
			if countDone >= len(r.jobs) {
				// All jobs are done, it's time to exit the loop
				break loop
			}
		case <-ctx.Done():
			// Shutdown signal received, exit the loop
			break loop
		}
	}

	// Wait for all dispatched jobs to finish before closing job event channels
	workerPool.StopAndWait() // Assuming your worker pool has a Wait method to wait for all dispatched jobs to complete

	// Safely close all job event channels after all work is done
	for _, job := range runningJobs {
		close(job.events)
	}
}

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
	fileSize := fileInfo.Size()
	r.fileSize = fileSize

	// Starting from the end of the file, read backwards
	for fileSize > 0 {
		// Move back to read the four-byte length at the end of the compressed event block
		offset := fileSize - 4
		_, err := file.Seek(offset, io.SeekStart)
		if err != nil {
			panic(fmt.Sprintf("failed to seek in file: %v", err))
		}

		// Read the four-byte length, big endian
		lengthBytes := make([]byte, 4)
		_, err = file.Read(lengthBytes)
		if err != nil {
			panic(fmt.Sprintf("failed to read event size: %v", err))
		}
		length := (int(lengthBytes[0]) << 24) + (int(lengthBytes[1]) << 16) + (int(lengthBytes[2]) << 8) + int(lengthBytes[3])

		// Validate length and ensure offset does not go beyond the file start
		if length <= 0 || length > int(fileSize-4) {
			fmt.Println("invalid block length or corrupted file")
			break
		}

		// Move back to read the compressed block payload
		offset -= int64(length)
		if offset < 0 {
			panic("offset calculated is beyond the file start, indicating a potential error in block length or file corruption")
		}
		_, err = file.Seek(offset, io.SeekStart)
		if err != nil {
			panic(fmt.Sprintf("failed to seek in file: %v", err))
		}

		// Read the compressed block payload
		compressedData := make([]byte, length)
		_, err = file.Read(compressedData)
		if err != nil {
			panic(fmt.Sprintf("failed to read compressed event payload: %v", err))
		}

		// Decompress the block payload
		gzr, err := gzip.NewReader(bytes.NewBuffer(compressedData))
		if err != nil {
			panic(fmt.Sprintf("failed to create gzip reader: %v", err))
		}
		decompressedData, err := io.ReadAll(gzr)
		if err != nil {
			panic(fmt.Sprintf("failed to decompress event payload: %v", err))
		}
		gzr.Close()

		// Unmarshal the block
		block := &ev.Block{}
		if err := proto.Unmarshal(decompressedData, block); err != nil {
			panic(fmt.Sprintf("failed to unmarshal block: %v", err))
		}

		// Send the events to the channel
		for _, e := range block.GetEvs() {
			r.events <- e
		}

		// Increment the event count
		r.currentReportEventCount += uint32(len(block.GetEvs()))

		// Update fileSize to the new offset for the next iteration
		fileSize = offset
	}

	// Close the events channel after reading all events
	close(r.events)

	// Update the last report event count
	r.lastReportEventCount = r.currentReportEventCount
}
