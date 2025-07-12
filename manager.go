package tache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync/atomic"

	nanoid "github.com/matoous/go-nanoid/v2"
	"github.com/xhofe/gsync"
)

// Manager is the manager of all tasks
type Manager[T Task] struct {
	rootTasks       gsync.MapOf[string, T]
	subtasks        gsync.MapOf[string, Task]
	toRunTaskQueue  gsync.QueueOf[T]
	workers         *WorkerPool
	opts            *Options
	debouncePersist func()
	running         atomic.Bool

	idGenerator func() string
	logger      *slog.Logger
}

// NewManager create a new manager
func NewManager[T Task](opts ...Option) *Manager[T] {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	m := &Manager[T]{
		workers: NewWorkerPool(options.Works),
		opts:    options,
		idGenerator: func() string {
			return nanoid.Must()
		},
		logger: options.Logger,
	}
	m.running.Store(options.Running)
	if m.opts.PersistPath != "" || (m.opts.PersistReadFunction != nil && m.opts.PersistWriteFunction != nil) {
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
			m.logger.Error("recover error", "error", err)
		}
	} else {
		m.debouncePersist = func() {}
	}
	return m
}

// Add a task to manager
func (m *Manager[T]) Add(task T) {
	ctx, cancel := context.WithCancel(context.Background())
	task.SetCtx(ctx)
	task.SetCancelFunc(cancel)
	task.SetPersist(m.debouncePersist)
	task.SetManager(m)
	if task.GetID() == "" {
		task.SetID(m.GenerateID())
	}
	if _, maxRetry := task.GetRetry(); maxRetry == 0 {
		task.SetRetry(0, m.opts.MaxRetry)
	}
	if sliceContains([]State{StateRunning}, task.GetState()) {
		task.SetState(StatePending)
	}
	if sliceContains([]State{StateCanceling}, task.GetState()) {
		task.SetState(StateCanceled)
		task.SetErr(context.Canceled)
	}
	if task.GetState() == StateFailing {
		task.SetState(StateFailed)
	}
	m.rootTasks.Store(task.GetID(), task)

	// Only add root tasks (tasks without parents) to the queue
	// Subtasks should be managed by their parent tasks
	if task.GetParent() == nil && !sliceContains([]State{StateSucceeded, StateCanceled, StateErrored, StateFailed}, task.GetState()) {
		m.toRunTaskQueue.Push(task)
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
	// if workers is full, return
	worker := m.workers.Get()
	if worker == nil {
		return
	}
	m.logger.Debug("got worker", slog.Int("id", worker.ID))
	task, err := m.toRunTaskQueue.Pop()
	// if cannot get task, return
	if err != nil {
		m.workers.Put(worker)
		return
	}
	m.logger.Debug("got task", slog.String("id", task.GetID()))
	go func() {
		defer func() {
			if task.GetState() == StateWaitingRetry {
				m.toRunTaskQueue.Push(task)
			}
			m.workers.Put(worker)
			m.next()
		}()
		if task.GetState() == StateCanceling {
			task.SetState(StateCanceled)
			task.SetErr(context.Canceled)
			return
		}
		if m.opts.Timeout != nil {
			ctx, cancel := context.WithTimeout(task.Ctx(), *m.opts.Timeout)
			defer cancel()
			task.SetCtx(ctx)
		}
		m.logger.Info("worker execute task", slog.Int("worker", worker.ID), slog.String("task", task.GetID()))
		worker.Execute(task)
	}()
}

// Wait wait all tasks done, just for test
func (m *Manager[T]) Wait() {
	for {
		tasks, running := m.toRunTaskQueue.Len(), m.workers.working
		if tasks == 0 && running == 0 {
			return
		}
		runtime.Gosched()
	}
}

// persist all tasks
func (m *Manager[T]) persist() error {
	if m.opts.PersistPath == "" && m.opts.PersistReadFunction == nil && m.opts.PersistWriteFunction == nil {
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
	if m.opts.PersistReadFunction != nil && m.opts.PersistWriteFunction != nil {
		err = m.opts.PersistWriteFunction(marshal)
		if err != nil {
			return err
		}
	}
	if m.opts.PersistPath != "" {
		// write to file
		err = os.WriteFile(m.opts.PersistPath, marshal, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// recover all tasks
func (m *Manager[T]) recover() error {
	var data []byte
	var err error
	if m.opts.PersistPath != "" {
		// read from file
		data, err = os.ReadFile(m.opts.PersistPath)
	} else if m.opts.PersistReadFunction != nil && m.opts.PersistWriteFunction != nil {
		data, err = m.opts.PersistReadFunction()
	} else {
		return nil
	}
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
			task.SetState(StateFailed)
			task.SetErr(fmt.Errorf("the task is interrupted and cannot be recovered"))
			m.rootTasks.Store(task.GetID(), task)
		}
	}
	return nil
}

// Cancel a task by ID
func (m *Manager[T]) Cancel(id string) {
	task := m.GetByIDAll(id)
	m.cancelTaskAndSubtasks(task)
	m.debouncePersist()
}

// cancelTaskAndSubtasks cancels a task and all its subtasks recursively
func (m *Manager[T]) cancelTaskAndSubtasks(task Task) {
	task.Cancel()

	// Cancel all subtasks,need? the context of subtasks will be canceled by the parent task
	// subtasks := task.GetSubtasks()
	// for _, subtask := range subtasks {
	// 	m.cancelTaskAndSubtasks(subtask)
	// }
}

// CancelAll cancel all tasks, just cancel all root tasks is enough
// because all subtasks will be canceled by their parent tasks
func (m *Manager[T]) CancelAll() {
	m.rootTasks.Range(func(key string, value T) bool {
		value.Cancel()
		return true
	})
	m.debouncePersist()
}

// CancelByCondition cancel tasks under specific condition given by a function
func (m *Manager[T]) CancelByCondition(condition func(task T) bool) {
	m.rootTasks.Range(func(key string, value T) bool {
		if condition(value) {
			value.Cancel()
		}
		return true
	})
}

// deprecated: use GetRootTasks instead
// GetAll get all tasks
func (m *Manager[T]) GetAll() []T {
	return m.GetRootTasks()
}

// GetRootTasks gets all root tasks (tasks without parents)
func (m *Manager[T]) GetRootTasks() []T {
	var tasks []T
	m.rootTasks.Range(func(key string, value T) bool {
		tasks = append(tasks, value)
		return true
	})
	return tasks
}

// GetByID get task by ID
func (m *Manager[T]) GetByID(id string) (T, bool) {
	return m.rootTasks.Load(id)
}

// GetByState get tasks by state
func (m *Manager[T]) GetByState(state ...State) []T {
	return m.GetByCondition(func(value T) bool {
		return sliceContains(state, value.GetState())
	})
}

// GetByCondition get tasks under specific condition given by a function
func (m *Manager[T]) GetByCondition(condition func(task T) bool) []T {
	var tasks []T
	m.rootTasks.Range(func(key string, value T) bool {
		if condition(value) {
			tasks = append(tasks, value)
		}
		return true
	})
	return tasks
}

// Remove a task by ID
func (m *Manager[T]) Remove(id string) {
	m.rootTasks.Delete(id)
	m.debouncePersist()
}

// RemoveAll remove all tasks
func (m *Manager[T]) RemoveAll() {
	tasks := m.GetAll()
	for _, task := range tasks {
		m.Remove(task.GetID())
	}
}

// RemoveByState remove tasks by state
func (m *Manager[T]) RemoveByState(state ...State) {
	tasks := m.GetByState(state...)
	for _, task := range tasks {
		m.Remove(task.GetID())
	}
}

// RemoveByCondition remove tasks under specific condition given by a function
func (m *Manager[T]) RemoveByCondition(condition func(task T) bool) {
	tasks := m.GetByCondition(condition)
	for _, task := range tasks {
		m.Remove(task.GetID())
	}
}

// Retry a task by ID
func (m *Manager[T]) Retry(id string) {
	if task, ok := m.rootTasks.Load(id); ok {
		task.SetState(StateWaitingRetry)
		task.SetErr(nil)
		task.SetRetry(0, m.opts.MaxRetry)
		m.toRunTaskQueue.Push(task)
		m.next()
		m.debouncePersist()
	}
}

// RetryAllFailed retry all failed tasks
func (m *Manager[T]) RetryAllFailed() {
	tasks := m.GetByState(StateFailed)
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

func (m *Manager[T]) SetWorkersNumActive(active int64) {
	oldActive := m.workers.numActive
	m.workers.SetNumActive(active)
	for i := 0; i < int(active-oldActive) && i < m.toRunTaskQueue.Len(); i++ {
		m.next()
	}
}

// GetTaskTrees gets all tasks in tree structure
func (m *Manager[T]) GetTaskTrees() []*TaskTree {
	rootTasks := m.GetRootTasks()
	trees := make([]*TaskTree, len(rootTasks))

	for i, rootTask := range rootTasks {
		trees[i] = m.buildTaskTree(rootTask)
	}

	return trees
}

// GetTreeByID gets the task tree by root task ID
func (m *Manager[T]) GetTreeByID(id string) *TaskTree {
	task := m.GetByIDAll(id)
	if task == nil {
		return nil
	}
	return m.buildTaskTree(task)
}

// buildTaskTree builds a task tree from a root task
func (m *Manager[T]) buildTaskTree(root Task) *TaskTree {
	tree := &TaskTree{
		Task:     root,
		Subtasks: make([]*TaskTree, 0),
	}

	subtasks := root.GetSubtasks()
	for _, subtask := range subtasks {
		childTree := m.buildTaskTree(subtask)
		tree.Subtasks = append(tree.Subtasks, childTree)
	}

	return tree
}

func (m *Manager[T]) GenerateID() string {
	return m.idGenerator()
}

func (m *Manager[T]) SaveSubTask(subtask Task) {
	m.subtasks.Store(subtask.GetID(), subtask)
}

// GetByIDAll gets the task by ID, including subtasks
func (m *Manager[T]) GetByIDAll(id string) Task {
	task, ok := m.rootTasks.Load(id)
	if ok {
		return task
	}
	subtask, ok := m.subtasks.Load(id)
	if ok {
		return subtask
	}
	return nil
}

// TaskTree represents a task in tree structure
type TaskTree struct {
	Task     Task        `json:"task"`
	Subtasks []*TaskTree `json:"subtasks"`
}
