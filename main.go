package main

import (
	"os"
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
			time.Sleep(time.Second * 5)
		}
	}()

	a := app.NewApp(&app.AppCfg{
		Filename:     filename,
		ReportRunner: r,
	})

	a.Start()
}
