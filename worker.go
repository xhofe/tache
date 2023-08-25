package tache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
)

type Worker[T Task] struct {
}

func (w Worker[T]) Run(task T) {
	onError := func(err error) {
		task.SetErr(err)
		if onFailed, ok := Task(task).(OnFailed); ok {
			task.SetStatus(FAILING)
			onFailed.OnFailed()
		}
		if errors.Is(err, context.Canceled) {
			task.SetStatus(CANCELED)
		} else {
			task.SetStatus(ERRORED)
		}
	}
	defer func() {
		if err := recover(); err != nil {
			log.Printf("error [%s] while run task [%d],stack trace:\n%s", err, task.GetID(), getCurrentGoroutineStack())
			onError(NewErr(fmt.Sprintf("panic: %v", err)))
		}
	}()
	task.SetStatus(RUNNING)
	err := task.Func()(task)
	if err != nil {
		onError(err)
		return
	}
	task.SetStatus(SUCCEEDED)
}

type WorkerPool[T Task] struct {
	workers chan Worker[T]
	working atomic.Int32
}

func NewWorkerPool[T Task](size int) *WorkerPool[T] {
	workers := make(chan Worker[T], size)
	for i := 0; i < size; i++ {
		workers <- Worker[T]{}
	}
	return &WorkerPool[T]{
		workers: workers,
	}
}

func (wp *WorkerPool[T]) Get() <-chan Worker[T] {
	return wp.workers
}

func (wp *WorkerPool[T]) Put(worker Worker[T]) {
	wp.workers <- worker
}
