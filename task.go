package tache

import "context"

// TaskBase is the base interface for all tasks
type TaskBase interface {
	// SetProgress sets the progress of the task
	SetProgress(progress float64)
	// GetProgress gets the progress of the task
	GetProgress() float64
	// SetState sets the state of the task
	SetState(state State)
	// GetState gets the state of the task
	GetState() State
	// GetID gets the ID of the task
	GetID() string
	// SetID sets the ID of the task
	SetID(id string)
	// SetErr sets the error of the task
	SetErr(err error)
	// GetErr gets the error of the task
	GetErr() error
	// SetCtx sets the context of the task
	SetCtx(ctx context.Context)
	// CtxDone gets the context done channel of the task
	CtxDone() <-chan struct{}
	// Cancel cancels the task
	Cancel()
	// Ctx gets the context of the task
	Ctx() context.Context
	// SetCancelFunc sets the cancel function of the task
	SetCancelFunc(cancelFunc context.CancelFunc)
	// GetRetry gets the retry of the task
	GetRetry() (int, int)
	// SetRetry sets the retry of the task
	SetRetry(retry int, maxRetry int)
	// Persist persists the task
	Persist()
	// SetPersist sets the persist function of the task
	SetPersist(persist func())
	// SetWorker sets the worker of the task
	SetWorker(worker *Worker)
	// GetWorker gets the worker of the task
	GetWorker() *Worker
	// AddSubtask adds a subtask to the current task
	AddSubtask(subtask Task)
	// GetSubtasks gets all subtasks of the current task
	GetSubtasks() []Task
	// GetParent gets the parent task of the current task
	GetParent() Task
	// SetParent sets the parent task of the current task
	SetParent(parent Task)
	// ExecuteSubtasks executes all subtasks and waits for them to complete
	ExecuteSubtasks() error
	// GetManager gets the manager of the current task
	GetManager() IManager
	// SetManager sets the manager of the current task
	SetManager(manager IManager)
	// GetSelf gets the self task of the current task
	GetSelf() Task
	// Wait waits for the task to complete, can be called multiple times
	Wait()
}

type NamedTask interface {
	GetName() string
}

type StatusTask interface {
	GetStatus() string
}

type Info interface {
	NamedTask
	StatusTask
}

// Task is the interface for all tasks
type Task interface {
	TaskBase
	Run() error
}

type TaskWithInfo interface {
	Task
	Info
}

type IManager interface {
	GenerateID() string
	SaveSubTask(subtask Task)
	GetByIDAll(id string) Task
}
