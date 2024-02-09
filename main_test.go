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
)

const (
	total = 100000000 // make 100M requests
	prod  = "https://lstn.swissinfo.ch"
)

func TestIntegration(t *testing.T) {
	_, local := os.LookupEnv("LOCAL")
	origin := prod
	if local {
		origin = "http://localhost:8080"
	}
	if local {
		testIntegration(origin, runtime.NumCPU()*16, total)
		return
	}
	testIntegration(origin, runtime.NumCPU()*128, total)
}

func testIntegration(origin string, concurrency, total int) {
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
	fmt.Printf("making %d requests to %s using %d workers\n", total, origin, concurrency)
	randUsr := strconv.FormatUint(uint64(rand.Uint32()), 10)
	go func() {
		for i := 0; i < total; i++ {
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
	for done < total && errs < total {
		err := <-out
		if err != nil {
			errs++
		} else {
			done++
			printProgress(done, errs, total)
		}
	}
	fmt.Printf("\n%d done and %d errors\n", done, errs)
}

func printProgress(done, errs, total int) {
	if done%10 == 0 || errs%10 == 0 {
		if done >= 1000000 {
			fmt.Printf("\r%.2fM requests done, and %d errors   ", float32(done)/float32(1000000), errs)
			return
		}
		if done >= 1000 {
			fmt.Printf("\r%.2fK requests done, and %d errors   ", float32(done)/float32(1000), errs)
			return
		}
		fmt.Printf("\r%d requests done, and %d errors   ", done, errs)
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
