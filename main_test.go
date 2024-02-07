package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

const (
	total         = 1000000 // make 1M requests
	originDefault = "https://lstn.swissinfo.ch"
)

func TestIntegration(t *testing.T) {
	origin := os.Getenv("LSTN_ORIGIN")
	if origin == "" {
		fmt.Println("LSTN_ORIGIN undefined, using default", originDefault)
		origin = originDefault
	}
	testIntegration(origin, runtime.NumCPU()*32, total)
}

func testIntegration(origin string, concurrency, total int) {
	jobs := make(chan *http.Request)
	wg := &sync.WaitGroup{}
	// start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int, wg *sync.WaitGroup) {
			count := 0
			for j := range jobs {
				_, err := http.DefaultClient.Do(j)
				if err != nil {
					panic(err)
				}
				count++
				if count%100 == 0 && i == 0 {
					done := float64(count*concurrency) / float64(total)
					fmt.Printf("worker %d made %d requests, done: %.2f\n", i, count, done)
				}
			}
			fmt.Printf("worker %d done\n", i)
			wg.Done()
		}(i, wg)
	}
	// get test cids
	cids := readFileLines(".testdata/cids.txt")
	// make requests
	fmt.Printf("making %d requests to %s using %d workers\n", total, origin, concurrency)
	randUsr := strconv.FormatUint(uint64(rand.Uint32()), 10)
	for i := 0; i < total; i++ {
		randSess := strconv.FormatUint(uint64(rand.Uint32()), 10)
		r, _ := http.NewRequest("POST", origin, nil)
		r.Header.Set("X_TYPE", "LOAD")
		r.Header.Set("X_USR", randUsr)
		r.Header.Set("X_SESS", randSess)
		r.Header.Set("X_CID", cids[rand.Intn(len(cids))])
		jobs <- r
	}
	close(jobs)
	// wait for workers to finish
	wg.Wait()
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
