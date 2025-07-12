package tache

import (
	"context"
	"sync"
)

// Base is the base struct for all tasks to implement TaskBase interface
type Base struct {
	ID       string `json:"id"`
	State    State  `json:"state"`
	Retry    int    `json:"retry"`
	MaxRetry int    `json:"max_retry"`

	progress   float64            `json:"-"`
	worker     *Worker            `json:"-"`
	err        error              `json:"-"`
	ctx        context.Context    `json:"-"`
	cancel     context.CancelFunc `json:"-"`
	persist    func()             `json:"-"`
	subtasks   []Task             `json:"-"`
	parent     Task               `json:"-"`
	subtasksMu sync.RWMutex       `json:"-"`
	manager    IManager           `json:"-"`
}

func (b *Base) SetProgress(progress float64) {
	b.progress = progress
	b.Persist()
}

func (b *Base) GetProgress() float64 {
	return b.progress
}

func (b *Base) SetState(state State) {
	b.State = state
	b.Persist()
}

func (b *Base) GetState() State {
	return b.State
}

func (b *Base) GetID() string {
	return b.ID
}

func (b *Base) SetID(id string) {
	b.ID = id
	b.Persist()
}

func (b *Base) SetErr(err error) {
	b.err = err
	b.Persist()
}

func (b *Base) GetErr() error {
	return b.err
}

func (b *Base) CtxDone() <-chan struct{} {
	return b.Ctx().Done()
}

func (b *Base) SetCtx(ctx context.Context) {
	b.ctx = ctx
}

func (b *Base) SetCancelFunc(cancelFunc context.CancelFunc) {
	b.cancel = cancelFunc
}

func (b *Base) GetRetry() (int, int) {
	return b.Retry, b.MaxRetry
}

func (b *Base) SetRetry(retry int, maxRetry int) {
	b.Retry, b.MaxRetry = retry, maxRetry
}

func (b *Base) Cancel() {
	b.SetState(StateCanceling)
	b.cancel()
}

func (b *Base) Ctx() context.Context {
	return b.ctx
}

func (b *Base) Persist() {
	if b.persist != nil {
		b.persist()
	}
}

func (b *Base) SetPersist(persist func()) {
	b.persist = persist
}

func (b *Base) SetWorker(worker *Worker) {
	b.worker = worker
}

func (b *Base) GetWorker() *Worker {
	return b.worker
}

func (b *Base) AddSubtask(subtask Task) {
	b.subtasksMu.Lock()
	defer b.subtasksMu.Unlock()

	// Note: Parent-child relationship should be set by the calling task
	// subtask.SetParent(parentTask) should be called by the parent task
	b.subtasks = append(b.subtasks, subtask)

	// Inherit context from parent
	if b.ctx != nil {
		newCtx, cancel := context.WithCancel(b.ctx)
		subtask.SetCtx(newCtx)
		subtask.SetCancelFunc(cancel)
	}
	// if id is not set, set it to the subtask id
	if subtask.GetID() == "" {
		subtask.SetID(b.manager.GenerateID())
	}
	// Set persist function
	subtask.SetPersist(b.persist)
	// Set parent
	subtask.SetParent(b.GetSelf())
	// Set manager
	subtask.SetManager(b.manager)
	// Save subtask to manager
	b.manager.SaveSubTask(subtask)
	// Persist
	b.Persist()
}

func (b *Base) GetSubtasks() []Task {
	b.subtasksMu.RLock()
	defer b.subtasksMu.RUnlock()

	result := make([]Task, len(b.subtasks))
	copy(result, b.subtasks)
	return result
}

func (b *Base) GetParent() Task {
	return b.parent
}

func (b *Base) SetParent(parent Task) {
	b.parent = parent
}

func (b *Base) ExecuteSubtasks() error {
	b.subtasksMu.RLock()
	subtasks := make([]Task, len(b.subtasks))
	copy(subtasks, b.subtasks)
	b.subtasksMu.RUnlock()

	// Execute all subtasks sequentially
	for _, subtask := range subtasks {
		// Skip if subtask is already completed or canceled
		if sliceContains([]State{StateSucceeded, StateCanceled, StateErrored, StateFailed}, subtask.GetState()) {
			continue
		}

		// Check if subtask is being canceled
		if subtask.GetState() == StateCanceling {
			subtask.SetState(StateCanceled)
			subtask.SetErr(context.Canceled)
			continue
		}

		// Execute subtask with proper error handling and hooks
		b.worker.Execute(subtask)
		if subtask.GetErr() != nil {
			return subtask.GetErr()
		}
	}

	return nil
}

func (b *Base) GetManager() IManager {
	return b.manager
}

func (b *Base) SetManager(manager IManager) {
	b.manager = manager
}

func (b *Base) GetSelf() Task {
	return b.GetManager().GetByIDAll(b.GetID())
}

var _ TaskBase = (*Base)(nil)
