package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/swissinfo-ch/lstn/report"
)

const (
	prod = "https://lstn.swissinfo.ch"
)

func TestIntegration(t *testing.T) {
	_, local := os.LookupEnv("LOCAL")
	origin := prod
	if local {
		origin = "http://localhost:8080"
	}
	count := 100000000 // 100M
	countEnv, ok := os.LookupEnv("COUNT")
	if ok {
		var err error
		count, err = strconv.Atoi(countEnv)
		if err != nil {
			panic(err)
		}
	}
	if local {
		// 2 is optimal on my machine, guessing it's pretty good
		testIntegration(origin, 2, count)
		return
	}
	// this works well for me
	testIntegration(origin, runtime.NumCPU()*128, count)
}

func testIntegration(origin string, concurrency, count int) {
	jobs := make(chan *http.Request, 50)
	out := make(chan *error, 50)
	// start workers
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			for j := range jobs {
				_, err := http.DefaultClient.Do(j)
				if err != nil {
					err := fmt.Errorf("worker %d error: %v\n", i, err)
					out <- &err
					continue
				}
				out <- nil
			}
		}(i)
	}
	// get test cids
	cids := readFileLines(".testdata/cids.txt")
	// make requests
	fmt.Printf("making %d requests to %s using %d workers\n", count, origin, concurrency)
	randUsr := strconv.FormatUint(uint64(rand.Uint32()), 10)
	go func() {
		for i := 0; i < count; i++ {
			randSess := strconv.FormatUint(uint64(rand.Uint32()), 10)
			r, _ := http.NewRequest("POST", origin, nil)
			r.Header.Set("TYPE", "LOAD")
			r.Header.Set("USR", randUsr)
			r.Header.Set("SESS", randSess)
			r.Header.Set("CID", cids[rand.Intn(len(cids))])
			jobs <- r
		}
		close(jobs)
	}()
	// collect results
	errs := 0
	done := 0
	tStart := time.Now()
	for done < count && errs < count {
		err := <-out
		if err != nil {
			errs++
		} else {
			done++
			printProgress(done, errs)
		}
	}
	duration := time.Since(tStart)
	evPerSec := report.FmtCount(uint32(float64(done) / duration.Seconds()))
	fmt.Printf("\n%d done and %d errors in %s at %s ev/s\n", done, errs, duration.String(), evPerSec)
}

func printProgress(done, errs int) {
	if done%10 == 0 || errs%10 == 0 {
		defer fmt.Print("\033[0K") // flush line
		if done >= 1000000 {
			fmt.Printf("\r%.2fM requests done, and %d errors", float32(done)/float32(1000000), errs)
			return
		}
		if done >= 1000 {
			fmt.Printf("\r%.2fK requests done, and %d errors", float32(done)/float32(1000), errs)
			return
		}
		fmt.Printf("\r%d requests done, and %d errors", done, errs)
	}
}

func readFileLines(fileName string) []string {
	file, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return lines
}
