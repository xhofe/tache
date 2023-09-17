package tache

import "context"

const (
	StatusPending = iota
	StatusRunning
	StatusSucceeded
	StatusCanceling
	StatusCanceled
	StatusFailing
	StatusFailed
	StatusErrored
	StatusWaitingRetry
)

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
	GetRetry() int
	SetRetry(retry int)
}

type Task interface {
	TaskBase
	Run() error
}

type Recover interface {
	Recover()
}

type OnFailed interface {
	OnFailed()
}
