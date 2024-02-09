package report

import (
	"bytes"
	"compress/gzip"
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
			fmt.Printf("\r%s reporting took %v for %s evs",
				r.lastReportTime.Format(time.RFC3339),
				r.lastReportDuration,
				FmtCount(r.currentReportEventCount))
			fmt.Print("\033[0K") // flush line
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
	// TUNING: 2024-02-09
	// r.events chan buffer size 10000 seems optimal
	r.events = make(chan *ev.Ev, 10000)
	for jobName, job := range r.jobs {
		// TUNING: 2024-02-09
		// job events chan buffer size 2 seems optimal
		job.events = make(chan *ev.Ev, 2)
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
	emptyChanCount := 0
loop:
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
			if countDone >= len(r.jobs) {
				break loop
			}
		// TUNING: events & jobDone channels empty check
		case <-time.After(time.Microsecond * 100):
			emptyChanCount++
		}
	}
	if emptyChanCount > 0 {
		fmt.Println("emptyChanCount", emptyChanCount)
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
