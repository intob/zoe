package worker

import (
	"fmt"
	"sync"
)

// Task is a function that performs a job.
type Task func()

// Pool manages a pool of workers to execute tasks.
type Pool struct {
	tasks        chan Task
	wg           sync.WaitGroup
	numOfWorkers int
	mu           sync.Mutex
	shuttingDown bool
}

// NewWorkerPool creates a new WorkerPool.
func NewPool(numOfWorkers int) *Pool {
	return &Pool{
		numOfWorkers: numOfWorkers,
	}
}

// Start initializes the workers and makes them ready to receive tasks.
func (wp *Pool) Start() {
	wp.tasks = make(chan Task)
	for i := 0; i < wp.numOfWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// StopAndWait signals the workers to stop and waits for all of them to finish.
func (p *Pool) StopAndWait() {
	p.mu.Lock()
	p.shuttingDown = true
	close(p.tasks)
	p.mu.Unlock()
	p.wg.Wait() // Wait for all worker goroutines to finish
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
