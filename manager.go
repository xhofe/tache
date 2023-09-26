package tache

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/xhofe/gsync"
)

// Manager is the manager of all tasks
type Manager[T Task] struct {
	tasks           gsync.MapOf[int64, T]
	queue           gsync.QueueOf[T]
	curID           atomic.Int64
	workers         *WorkerPool[T]
	opts            *Options
	debouncePersist func()
}

// NewManager create a new manager
func NewManager[T Task](opts ...Option) *Manager[T] {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	m := &Manager[T]{
		workers: NewWorkerPool[T](options.Works),
		opts:    options,
	}
	if m.opts.PersistPath != "" {
		m.debouncePersist = func() {
			_ = m.persist()
		}
		if m.opts.PersistDebounce != nil {
			m.debouncePersist = newDebounce(func() {
				_ = m.persist()
			}, *m.opts.PersistDebounce)
		}
		err := m.recover()
		if err != nil {
			// TODO: log?
		}
	}
	return m
}

// Add a task to manager
func (m *Manager[T]) Add(task T) {
	ctx, cancel := context.WithCancel(context.Background())
	task.SetCtx(ctx)
	task.SetCancelFunc(func() {
		cancel()
	})
	task.SetPersist(m.debouncePersist)
	task.SetID(m.curID.Add(1))
	if task.GetStatus() == StatusCanceling {
		task.SetStatus(StatusCanceled)
		task.SetErr(context.Canceled)
	}
	if task.GetStatus() == StatusFailing {
		task.SetStatus(StatusFailed)
	}
	m.tasks.Store(task.GetID(), task)
	if !sliceContains([]Status{StatusSucceeded, StatusCanceled, StatusErrored, StatusFailed}, task.GetStatus()) {
		m.queue.Push(task)
	}
	m.next()
}

// get next task from queue and execute it
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

// needRetry judge whether the task need retry
func (m *Manager[T]) needRetry(task T) bool {
	// if task is not recoverable, return false
	if !IsRecoverable(task.GetErr()) {
		return false
	}
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

// persist all tasks
func (m *Manager[T]) persist() error {
	if m.opts.PersistPath == "" {
		return nil
	}
	// serialize tasks
	tasks := m.GetAll()
	var toPersist []T
	for _, task := range tasks {
		// only persist task which is not persistable or persistable and need persist
		if p, ok := Task(task).(Persistable); !ok || p.Persistable() {
			toPersist = append(toPersist, task)
		}
	}
	marshal, err := json.Marshal(toPersist)
	if err != nil {
		return err
	}
	// write to file
	err = os.WriteFile(m.opts.PersistPath, marshal, 0644)
	if err != nil {
		return err
	}
	return nil
}

// recover all tasks
func (m *Manager[T]) recover() error {
	if m.opts.PersistPath == "" {
		return nil
	}
	// read from file
	data, err := os.ReadFile(m.opts.PersistPath)
	if err != nil {
		return err
	}
	// deserialize tasks
	var tasks []T
	err = json.Unmarshal(data, &tasks)
	if err != nil {
		return err
	}
	// add tasks
	for _, task := range tasks {
		// only recover task which is not recoverable or recoverable and need recover
		if r, ok := Task(task).(Recoverable); !ok || r.Recoverable() {
			m.Add(task)
		}
	}
	return nil
}

// Cancel a task by id
func (m *Manager[T]) Cancel(id int64) {
	if task, ok := m.tasks.Load(id); ok {
		task.Cancel()
	}
}

// CancelAll cancel all tasks
func (m *Manager[T]) CancelAll() {
	m.tasks.Range(func(key int64, value T) bool {
		value.Cancel()
		return true
	})
}

// GetAll get all tasks
func (m *Manager[T]) GetAll() []T {
	var tasks []T
	m.tasks.Range(func(key int64, value T) bool {
		tasks = append(tasks, value)
		return true
	})
	return tasks
}

// GetByID get task by id
func (m *Manager[T]) GetByID(id int64) (T, bool) {
	return m.tasks.Load(id)
}

// GetByStatus get tasks by status
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

// Remove a task by id
func (m *Manager[T]) Remove(id int64) {
	m.tasks.Delete(id)
}

// RemoveAll remove all tasks
func (m *Manager[T]) RemoveAll() {
	tasks := m.GetAll()
	for _, task := range tasks {
		m.Remove(task.GetID())
	}
}

// RemoveByStatus remove tasks by status
func (m *Manager[T]) RemoveByStatus(status ...Status) {
	tasks := m.GetByStatus(status...)
	for _, task := range tasks {
		m.Remove(task.GetID())
	}
}
