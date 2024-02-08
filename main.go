package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/swissinfo-ch/lstn/app"
	"github.com/swissinfo-ch/lstn/ev"
	"github.com/swissinfo-ch/lstn/report"
)

func main() {
	filename := "events"
	filenameEnv, ok := os.LookupEnv("LSTN_EVENTS_FILE")
	if ok {
		filename = filenameEnv
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	file.Close()

	minReportInterval := time.Second * 60
	minReportIntervalEnv, ok := os.LookupEnv("LSTN_MIN_REPORT_INTERVAL")
	if ok {
		var err error
		minReportInterval, err = time.ParseDuration(minReportIntervalEnv)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("setting min report interval to", minReportInterval)

	runnerCfg := &report.RunnerCfg{
		Filename: filename,
		Jobs: map[string]*report.Job{
			"views-last30d-cutoff10": {
				Report: &report.Views{
					Cutoff:        10,
					EstimatedSize: 1000,
					Filter: func(e *ev.Ev) bool {
						return e.EvType == ev.EvType_LOAD && report.YoungerThan(e, time.Hour*24*30)
					},
				},
			},
			"top10-last30d": {
				Report: &report.Top{
					N: 10,
					Filter: func(e *ev.Ev) bool {
						return e.EvType == ev.EvType_LOAD && report.YoungerThan(e, time.Hour*24*30)
					},
				},
			},
			"subset-last7d-max3": {
				Report: &report.Subset{
					Limit: 3,
					Filter: func(e *ev.Ev) bool {
						return e.EvType == ev.EvType_LOAD && report.YoungerThan(e, time.Hour*24*7)
					},
				},
			},
		},
	}
	reportsRunner := report.NewRunner(runnerCfg)
	go func() {
		lastTimeLogged := time.Time{}
		for {
			tStart := time.Now()
			reportsRunner.Run()
			tEnd := time.Now()
			// occasionally log report running time
			if time.Since(lastTimeLogged) > time.Minute*5 {
				fmt.Printf("reporting took %v\n", time.Since(tStart))
				lastTimeLogged = time.Now()
			}
			// limit report running rate
			if tEnd.Sub(tStart) < minReportInterval {
				time.Sleep((minReportInterval) - tEnd.Sub(tStart))
			}
		}
	}()

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
	})

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
