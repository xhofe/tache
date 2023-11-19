package tache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jaevor/go-nanoid"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/xhofe/gsync"
)

// Manager is the manager of all tasks
type Manager[T Task] struct {
	tasks           gsync.MapOf[string, T]
	queue           gsync.QueueOf[T]
	workers         *WorkerPool[T]
	opts            *Options
	debouncePersist func()
	running         atomic.Bool

	idGenerator func() string
}

// NewManager create a new manager
func NewManager[T Task](opts ...Option) *Manager[T] {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	nanoID, err := nanoid.Standard(21)
	if err != nil {
		panic(err)
	}
	m := &Manager[T]{
		workers:     NewWorkerPool[T](options.Works),
		opts:        options,
		idGenerator: nanoID,
	}
	m.running.Store(options.Running)
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
	task.SetCancelFunc(cancel)
	task.SetPersist(m.debouncePersist)
	if task.GetID() == "" {
		task.SetID(m.idGenerator())
	}
	if _, maxRetry := task.GetRetry(); maxRetry == 0 {
		task.SetRetry(0, m.opts.MaxRetry)
	}
	if sliceContains([]Status{StatusRunning}, task.GetStatus()) {
		task.SetStatus(StatusPending)
	}
	if sliceContains([]Status{StatusCanceling}, task.GetStatus()) {
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
	m.debouncePersist()
	m.next()
}

// get next task from queue and execute it
func (m *Manager[T]) next() {
	// if manager is not running, return
	if !m.running.Load() {
		return
	}
	task, err := m.queue.Pop()
	// if cannot get task, return
	if err != nil {
		return
	}
	// if workers is full, return
	worker := m.workers.Get()
	if worker == nil {
		return
	}
	go func() {
		defer func() {
			if task.GetStatus() == StatusWaitingRetry {
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
		} else {
			task.SetStatus(StatusFailed)
			task.SetErr(fmt.Errorf("the task is interrupted and cannot be recovered"))
			m.tasks.Store(task.GetID(), task)
		}
	}
	return nil
}

// Cancel a task by ID
func (m *Manager[T]) Cancel(id string) {
	if task, ok := m.tasks.Load(id); ok {
		task.Cancel()
		m.debouncePersist()
	}
}

// CancelAll cancel all tasks
func (m *Manager[T]) CancelAll() {
	m.tasks.Range(func(key string, value T) bool {
		value.Cancel()
		return true
	})
	m.debouncePersist()
}

// GetAll get all tasks
func (m *Manager[T]) GetAll() []T {
	var tasks []T
	m.tasks.Range(func(key string, value T) bool {
		tasks = append(tasks, value)
		return true
	})
	return tasks
}

// GetByID get task by ID
func (m *Manager[T]) GetByID(id string) (T, bool) {
	return m.tasks.Load(id)
}

// GetByStatus get tasks by status
func (m *Manager[T]) GetByStatus(status ...Status) []T {
	var tasks []T
	m.tasks.Range(func(key string, value T) bool {
		if sliceContains(status, value.GetStatus()) {
			tasks = append(tasks, value)
		}
		return true
	})
	return tasks
}

// Remove a task by ID
func (m *Manager[T]) Remove(id string) {
	m.tasks.Delete(id)
	m.debouncePersist()
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

// Retry a task by ID
func (m *Manager[T]) Retry(id string) {
	if task, ok := m.tasks.Load(id); ok {
		task.SetStatus(StatusWaitingRetry)
		task.SetErr(nil)
		task.SetRetry(0, m.opts.MaxRetry)
		m.queue.Push(task)
		m.debouncePersist()
	}
}

// RetryAllFailed retry all failed tasks
func (m *Manager[T]) RetryAllFailed() {
	tasks := m.GetByStatus(StatusFailed)
	for _, task := range tasks {
		m.Retry(task.GetID())
	}
}

// Start manager
func (m *Manager[T]) Start() {
	m.running.Store(true)
	m.next()
}

// Pause manager
func (m *Manager[T]) Pause() {
	m.running.Store(false)
}
