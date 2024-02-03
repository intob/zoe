package main

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"testing"
)

const count = 10000 // make 10k requests

func testIntegration(url string, concurrency, count int) {
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
				if count%100 == 0 {
					fmt.Printf("worker %d made %d requests\n", i, count)
				}
			}
			fmt.Printf("worker %d done\n", i)
			wg.Done()
		}(i, wg)
		fmt.Printf("worker %d started\n", i)
	}
	for i := 0; i < count; i++ {
		r, _ := http.NewRequest("POST", url, nil)
		r.Header.Set("LSTN_TYPE", "LOAD")
		r.Header.Set("LSTN_USR", "12345")
		r.Header.Set("LSTN_SESS", "23456")
		r.Header.Set("LSTN_CID", "34567")
		jobs <- r
	}
	close(jobs)
	wg.Wait()
}

func TestOnFly(t *testing.T) {
	testIntegration("https://lstn.fly.dev", runtime.NumCPU()*8, count)
}

func TestLocal(t *testing.T) {
	testIntegration("http://127.0.0.1:8080", runtime.NumCPU(), count)
}
