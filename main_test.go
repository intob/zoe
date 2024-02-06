package main

import (
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

func testIntegration(origin string, concurrency, total int) {
	jobs := make(chan *http.Request)
	wg := &sync.WaitGroup{}
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
		fmt.Printf("worker %d started\n", i)
	}
	usr := strconv.FormatUint(uint64(rand.Uint32()), 10)
	for i := 0; i < total; i++ {
		r, _ := http.NewRequest("POST", origin, nil)
		r.Header.Set("X_TYPE", "LOAD")
		r.Header.Set("X_USR", usr)
		r.Header.Set("X_SESS", strconv.FormatUint(uint64(rand.Uint32()), 10))
		r.Header.Set("X_CID", strconv.FormatUint(uint64(i), 10))
		jobs <- r
	}
	close(jobs)
	wg.Wait()
}

func TestIntegration(t *testing.T) {
	origin := os.Getenv("LSTN_ORIGIN")
	if origin == "" {
		fmt.Println("LSTN_ORIGIN undefined, using default", originDefault)
		origin = originDefault
	}
	testIntegration(origin, runtime.NumCPU()*32, total)
}
