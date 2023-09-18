package tache

import (
	"context"
	"runtime"
	"sync/atomic"

	"github.com/xhofe/gsync"
)

type Manager[T Task] struct {
	tasks   gsync.MapOf[int64, T]
	queue   gsync.QueueOf[T]
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
		workers: NewWorkerPool[T](options.Works),
		opts:    options,
	}
	return m
}

func (m *Manager[T]) Add(task T) {
	ctx, cancel := context.WithCancel(context.Background())
	task.SetCtx(ctx)
	task.SetCancelFunc(func() {
		cancel()
	})
	task.SetID(m.curID.Add(1))
	m.tasks.Store(task.GetID(), task)
	m.queue.Push(task)
	m.next()
}

func (m *Manager[T]) next() {
	if m.queue.Len() == 0 {
		return
	}
	worker := m.workers.Get()
	if worker == nil {
		return
	}
	task := m.queue.MustPop()
	go func() {
		defer func() {
			if m.needRetry(task) {
				m.queue.Push(task)
			}
			m.workers.Put(worker)
			m.next()
		}()
		if task.GetStatus() == StatusCanceling {
			task.SetStatus(StatusCanceled)
			task.SetErr(context.Canceled)
			return
		}
		if m.opts.Timeout != nil {
			ctx, cancel := context.WithTimeout(task.Ctx(), *m.opts.Timeout)
			defer cancel()
			task.SetCtx(ctx)
		}
		worker.Execute(task)
	}()
}

func (m *Manager[T]) needRetry(task T) bool {
	if sliceContains([]Status{StatusErrored, StatusFailed}, task.GetStatus()) {
		if task.GetRetry() < m.opts.Retry {
			task.SetRetry(task.GetRetry() + 1)
			task.SetStatus(StatusWaitingRetry)
			return true
		}
	}
	return false
}

// Wait wait all tasks done, just for test
func (m *Manager[T]) Wait() {
	for {
		tasks, running := m.queue.Len(), m.workers.working.Load()
		if tasks == 0 && running == 0 {
			return
		}
		runtime.Gosched()
	}
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

func (m *Manager[T]) GetByStatus(status ...Status) []T {
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

func (m *Manager[T]) RemoveByStatus(status ...Status) {
	tasks := m.GetByStatus(status...)
	for _, task := range tasks {
		m.Remove(task.GetID())
	}
}
