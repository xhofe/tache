package tache

import "context"

const (
	PENDING = iota
	RUNNING
	SUCCEEDED
	CANCELING
	CANCELED
	FAILING
	ERRORED
)

type Func func(task Task) error

type TaskBase interface {
	SetProgress(progress float64)
	GetProgress() float64
	SetStatus(status int)
	GetStatus() int
	GetID() int64
	SetID(id int64)
	SetErr(err error)
	GetErr() error
	SetCtx(ctx context.Context)
	CtxDone() <-chan struct{}
	Cancel()
	SetCancelFunc(cancelFunc context.CancelFunc)
}

type Task interface {
	TaskBase
	Func() Func
}

type OnFailed interface {
	OnFailed()
}
