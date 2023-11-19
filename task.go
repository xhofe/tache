package tache

import "context"

// TaskBase is the base interface for all tasks
type TaskBase interface {
	// SetProgress sets the progress of the task
	SetProgress(progress float64)
	// GetProgress gets the progress of the task
	GetProgress() float64
	// SetStatus sets the status of the task
	SetStatus(status Status)
	// GetStatus gets the status of the task
	GetStatus() Status
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
}

// Task is the interface for all tasks
type Task interface {
	TaskBase
	Run() error
}
