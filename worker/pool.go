package worker

import (
	"fmt"
	"sync"
)

// Pool manages a pool of workers that execute tasks.
type Pool struct {
	tasks        chan func()
	wg           sync.WaitGroup
	mu           sync.Mutex
	shuttingDown bool
}

// NewWorkerPool creates a new worker pool.
func NewPool(numOfWorkers int) *Pool {
	wp := &Pool{
		tasks: make(chan func()),
	}
	for i := 0; i < numOfWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
	return wp
}

// StopAndWait signals the workers to stop, and waits for all to finish.
func (p *Pool) StopAndWait() {
	p.mu.Lock()
	p.shuttingDown = true
	close(p.tasks)
	p.mu.Unlock()
	p.wg.Wait()
}

// Dispatch sends a task to the worker pool.
func (p *Pool) Dispatch(task func()) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.shuttingDown {
		return fmt.Errorf("worker pool is shutting down")
	}
	select {
	case p.tasks <- task:
		return nil
	default:
		return fmt.Errorf("worker pool task queue is full")
	}
}

// worker is the function run by each worker goroutine.
func (wp *Pool) worker() {
	defer wp.wg.Done()
	for task := range wp.tasks {
		if task != nil {
			task()
		}
	}
}
