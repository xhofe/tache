package tache

import "context"

type TaskBase interface {
	SetProgress(progress float64)
	GetProgress() float64
	SetStatus(status Status)
	GetStatus() Status
	GetID() int64
	SetID(id int64)
	SetErr(err error)
	GetErr() error
	SetCtx(ctx context.Context)
	CtxDone() <-chan struct{}
	Cancel()
	Ctx() context.Context
	SetCancelFunc(cancelFunc context.CancelFunc)
	GetRetry() int
	SetRetry(retry int)
	Persist()
	SetPersist(persist func())
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
