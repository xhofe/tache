package tache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
)

// Worker is the worker to execute task
type Worker[T Task] struct {
}

// Execute executes the task
func (w Worker[T]) Execute(task T) {
	onError := func(err error) {
		task.SetErr(err)
		if onFailed, ok := Task(task).(OnFailed); ok {
			task.SetStatus(StatusFailing)
			onFailed.OnFailed()
			task.SetStatus(StatusFailed)
		}
		if errors.Is(err, context.Canceled) {
			task.SetStatus(StatusCanceled)
		} else {
			task.SetStatus(StatusErrored)
		}
	}
	defer func() {
		if err := recover(); err != nil {
			log.Printf("error [%s] while run task [%d],stack trace:\n%s", err, task.GetID(), getCurrentGoroutineStack())
			onError(NewErr(fmt.Sprintf("panic: %v", err)))
		}
	}()
	task.SetStatus(StatusRunning)
	err := task.Run()
	if err != nil {
		onError(err)
		return
	}
	task.SetStatus(StatusSucceeded)
	if onSucceeded, ok := Task(task).(OnSucceeded); ok {
		onSucceeded.OnSucceeded()
	}
	task.SetErr(nil)
}

// WorkerPool is the pool of workers
type WorkerPool[T Task] struct {
	working atomic.Int64
	workers chan *Worker[T]
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool[T Task](size int) *WorkerPool[T] {
	workers := make(chan *Worker[T], size)
	for i := 0; i < size; i++ {
		workers <- &Worker[T]{}
	}
	return &WorkerPool[T]{
		workers: workers,
	}
}

// Get gets a worker from pool
func (wp *WorkerPool[T]) Get() *Worker[T] {
	select {
	case worker := <-wp.workers:
		wp.working.Add(1)
		return worker
	default:
		return nil
	}
}

// Put puts a worker back to pool
func (wp *WorkerPool[T]) Put(worker *Worker[T]) {
	wp.workers <- worker
	wp.working.Add(-1)
}
