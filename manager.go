package tache

import (
	"context"
	"sync/atomic"

	"github.com/xhofe/gsync"
)

type Manager[T Task] struct {
	tasks   gsync.MapOf[int64, T]
	curID   atomic.Int64
	workers *WorkerPool[T]
	opts    *Options
}

func NewManager[T Task](opts ...Option) *Manager[T] {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	m := &Manager[T]{
		workers: NewWorkerPool[T](options.Max),
		opts:    options,
	}
	return m
}

func (m *Manager[T]) Add(task T) {
	ctx, cancel := context.WithCancel(context.Background())
	task.SetCtx(ctx)
	task.SetCancelFunc(func() {
		task.SetStatus(CANCELING)
		cancel()
	})
	task.SetID(m.curID.Add(1))
	m.tasks.Store(task.GetID(), task)
	m.do(task)
}

func (m *Manager[T]) do(task T) {
	go func() {
		task.SetStatus(PENDING)
		select {
		case <-task.CtxDone():
			return
		case worker := <-m.workers.Get():
			worker.Run(task)
			m.workers.Put(worker)
		}
	}()
}

func (m *Manager[T]) Cancel(id int64) {
	if task, ok := m.tasks.Load(id); ok {
		task.Cancel()
	}
}

func (m *Manager[T]) CancelAll() {
	m.tasks.Range(func(key int64, value T) bool {
		value.Cancel()
		return true
	})
}

func (m *Manager[T]) GetAll() []T {
	var tasks []T
	m.tasks.Range(func(key int64, value T) bool {
		tasks = append(tasks, value)
		return true
	})
	return tasks
}

func (m *Manager[T]) GetByID(id int64) (T, bool) {
	return m.tasks.Load(id)
}

func (m *Manager[T]) GetByStatus(status ...int) []T {
	var tasks []T
	m.tasks.Range(func(key int64, value T) bool {
		if sliceContains(status, value.GetStatus()) {
			tasks = append(tasks, value)
		}
		return true
	})
	return tasks
}

func (m *Manager[T]) Remove(id int64) {
	m.tasks.Delete(id)
}

func (m *Manager[T]) RemoveAll() {
	tasks := m.GetAll()
	for _, task := range tasks {
		m.Remove(task.GetID())
	}
}

func (m *Manager[T]) RemoveByStatus(status ...int) {
	tasks := m.GetByStatus(status...)
	for _, task := range tasks {
		m.Remove(task.GetID())
	}
}
