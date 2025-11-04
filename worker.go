package tache

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
)

// Worker is the worker to execute task
type Worker struct {
	ID int
}

// Execute executes the task
func (w Worker) Execute(task Task) {
	task.SetWorker(&w)
	// Retry immediately in the same worker until success or max retry exhausted
	for {
		if isRetry(task) {
			task.SetState(StateBeforeRetry)
			if hook, ok := Task(task).(OnBeforeRetry); ok {
				hook.OnBeforeRetry()
			}
		}
		
		onError := func(err error) {
			task.SetErr(err)
			if errors.Is(err, context.Canceled) {
				task.SetState(StateCanceled)
			} else {
				task.SetState(StateErrored)
			}
			if !needRetry(task) {
				if hook, ok := Task(task).(OnFailed); ok {
					task.SetState(StateFailing)
					hook.OnFailed()
				}
				task.SetState(StateFailed)
			}
		}
		
		func() {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("error [%s] while run task [%s],stack trace:\n%s", err, task.GetID(), getCurrentGoroutineStack())
					onError(NewErr(fmt.Sprintf("panic: %v", err)))
				}
			}()
			
			task.SetState(StateRunning)
			err := task.Run()
			if err != nil {
				onError(err)
				return
			}
			
			task.SetState(StateSucceeded)
			if onSucceeded, ok := Task(task).(OnSucceeded); ok {
				onSucceeded.OnSucceeded()
			}
			task.SetErr(nil)
		}()
		
		state := task.GetState()
		if state == StateSucceeded || state == StateCanceled || state == StateFailed {
			return
		}
	}
}

// WorkerPool is the pool of workers
type WorkerPool struct {
	workers      *list.List
	workersMutex sync.Mutex
	working      int64
	numCreated   int64
	numActive    int64
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(size int) *WorkerPool {
	workers := list.New()
	for i := 0; i < size; i++ {
		workers.PushBack(&Worker{
			ID: i,
		})
	}
	return &WorkerPool{
		workers:    workers,
		numCreated: int64(size),
		numActive:  int64(size),
	}
}

// Get gets a worker from pool
func (wp *WorkerPool) Get() *Worker {
	wp.workersMutex.Lock()
	defer wp.workersMutex.Unlock()
	if wp.working >= wp.numActive {
		return nil
	}
	wp.working += 1
	if wp.workers.Len() > 0 {
		ret := wp.workers.Front().Value.(*Worker)
		wp.workers.Remove(wp.workers.Front())
		return ret
	}
	ret := &Worker{
		ID: int(wp.numCreated),
	}
	wp.numCreated += 1
	return ret
}

// Put puts a worker back to pool
func (wp *WorkerPool) Put(worker *Worker) {
	wp.workersMutex.Lock()
	defer wp.workersMutex.Unlock()
	wp.workers.PushBack(worker)
	wp.working -= 1
}

func (wp *WorkerPool) SetNumActive(active int64) {
	wp.workersMutex.Lock()
	defer wp.workersMutex.Unlock()
	wp.numActive = active
}
