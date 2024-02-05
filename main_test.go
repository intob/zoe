package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
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
	usr := strconv.FormatUint(uint64(rand.Uint32()), 10)
	cid := strconv.FormatUint(uint64(rand.Uint32()), 10)
	for i := 0; i < count; i++ {
		r, _ := http.NewRequest("POST", url, nil)
		r.Header.Set("X_TYPE", "LOAD")
		r.Header.Set("X_USR", usr)
		r.Header.Set("X_SESS", strconv.FormatUint(uint64(rand.Uint32()), 10))
		r.Header.Set("X_CID", cid)
		jobs <- r
	}
	close(jobs)
	wg.Wait()
}

func TestProd(t *testing.T) {
	testIntegration("https://lstn.swissinfo.ch", runtime.NumCPU()*8, count)
}

func TestLocal(t *testing.T) {
	testIntegration("http://127.0.0.1:8080", runtime.NumCPU(), count)
}
