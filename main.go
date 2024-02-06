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
	filenameEnv, ok := os.LookupEnv("EVENTS_FILE")
	if ok {
		filename = filenameEnv
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	file.Close()

	r := report.NewRunner(&report.RunnerCfg{
		Filename: filename,
		Jobs: []*report.Job{
			{
				Name: "views-last-30d",
				Report: &report.Views{
					Filter: func(e *ev.Ev) bool {
						return report.YoungerThan(e, time.Hour*24*30) &&
							e.EvType == ev.EvType_LOAD
					},
				},
			},
		},
	})
	go func() {
		for {
			r.Run()
			time.Sleep(time.Second * 2)
		}
	}()

	ctx := getCtx()

	a := app.NewApp(&app.AppCfg{
		Filename:     filename,
		ReportRunner: r,
		Ctx:          ctx,
	})
	go a.Start()

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
