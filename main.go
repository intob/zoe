package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/swissinfo-ch/zoe/app"
	"github.com/swissinfo-ch/zoe/ev"
	"github.com/swissinfo-ch/zoe/report"
)

func main() {
	// setup events file
	filename := "events"
	filenameEnv, ok := os.LookupEnv("ZOE_EVENTS_FILE")
	if ok {
		filename = filenameEnv
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	file.Close()
	fmt.Println("reading events from", filename)

	// setup block size
	blockSize := 10000
	blockSizeEnv, ok := os.LookupEnv("ZOE_BLOCK_SIZE")
	if ok {
		var err error
		blockSize, err = strconv.Atoi(blockSizeEnv)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("block size set to", blockSize)

	// setup worker pool size
	workerPoolSize := runtime.NumCPU()
	workerPoolSizeEnv, ok := os.LookupEnv("ZOE_WORKER_POOL_SIZE")
	if ok {
		var err error
		workerPoolSize, err = strconv.Atoi(workerPoolSizeEnv)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("worker pool size set to", workerPoolSize)

	// setup min report interval
	minReportInterval := time.Second * 5
	minReportIntervalEnv, ok := os.LookupEnv("ZOE_MIN_REPORT_INTERVAL")
	if ok {
		var err error
		minReportInterval, err = time.ParseDuration(minReportIntervalEnv)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("min report interval set to", minReportInterval)

	// setup report runner
	runnerCfg := &report.RunnerCfg{
		Filename:          filename,
		BlockSize:         blockSize,
		WorkerPoolSize:    workerPoolSize,
		MinReportInterval: minReportInterval,
		Jobs: map[string]*report.Job{
			"views-cutoff1000-last30d": {
				Report: &report.Views{
					Cutoff:        1000,
					EstimatedSize: 10000,
					MinEvTime: func() time.Time {
						return time.Now().Add(-time.Hour * 24 * 30)
					},
				},
			},
			"views-top10-last30d": {
				Report: &report.Top{
					N: 10,
					MinEvTime: func() time.Time {
						return time.Now().Add(-time.Hour * 24 * 30)
					},
				},
			},
			"subset-views-max10k": {
				Report: &report.Subset{
					Limit: 10000,
					Filter: func(e *ev.Ev) bool {
						return e.EvType == ev.EvType_LOAD
					},
				},
			},
		},
	}
	reportsRunner := report.NewRunner(runnerCfg)

	// setup app
	reportNames := make([]string, 0, len(runnerCfg.Jobs))
	for name := range runnerCfg.Jobs {
		reportNames = append(reportNames, name)
	}
	ctx := getCtx()
	app.NewApp(&app.AppCfg{
		Filename:     filename,
		ReportRunner: reportsRunner,
		ReportNames:  reportNames,
		Ctx:          ctx,
		BlockSize:    blockSize,
	})

	// wait for context to be done
	<-ctx.Done()
	fmt.Println("app shutting down")
}

// cancelOnKillSig cancels the context on os interrupt kill signal
func cancelOnKillSig(sigs chan os.Signal, cancel context.CancelFunc) {
	switch <-sigs {
	case syscall.SIGINT:
		fmt.Println("\nreceived SIGINT")
	case syscall.SIGTERM:
		fmt.Println("\nreceived SIGTERM")
	}
	cancel()
}

// getCtx returns a root context that awaits a kill signal from os
func getCtx() context.Context {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go cancelOnKillSig(sigs, cancel)
	return ctx
}
