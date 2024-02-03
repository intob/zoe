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
	a := app.NewApp(&app.AppCfg{
		Filename: filename,
	})
	go reports(filename, time.Second*10)
	a.Start()
}

func reports(filename string, every time.Duration) {
	r := &report.Runner{
		Filename: filename,
		Jobs: []*report.Job{
			{
				Name: "views last 30d",
				Report: &report.Views{
					Filter: func(e *ev.Ev) bool {
						return report.YoungerThan(e, time.Hour*24*30)
					},
				},
			},
		},
	}
	for {
		r.Run()
		time.Sleep(every)
	}
}
