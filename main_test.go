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

	"github.com/swissinfo-ch/zoe/report"
)

const (
	prod = "https://zoe.swissinfo.ch"
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
	fmt.Printf("making %s requests to %s using %d workers\n",
		report.FmtCount(uint32(count)), origin, concurrency)
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
	fmt.Printf("\n%s done and %s errors in %s at %s ev/s\n",
		report.FmtCount(uint32(done)), report.FmtCount(uint32(errs)), duration.String(), evPerSec)
}

func printProgress(done, errs int) {
	if done%10000 == 0 || errs%10 == 0 {
		fmt.Printf("\r%s requests done, and %s errors",
			report.FmtCount(uint32(done)), report.FmtCount(uint32(errs)))
		fmt.Print("\033[0K") // flush line
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
