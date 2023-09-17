package tache

import "context"

type Base struct {
	Progress   float64
	Status     int
	ID         int64
	Err        error
	Ctx        context.Context
	CancelFunc context.CancelFunc
	Retry      int
}

func (b *Base) SetProgress(progress float64) {
	b.Progress = progress
}

func (b *Base) GetProgress() float64 {
	return b.Progress
}

func (b *Base) SetStatus(status int) {
	b.Status = status
}

func (b *Base) GetStatus() int {
	return b.Status
}

func (b *Base) GetID() int64 {
	return b.ID
}

func (b *Base) SetID(id int64) {
	b.ID = id
}

func (b *Base) SetErr(err error) {
	b.Err = err
}

func (b *Base) GetErr() error {
	return b.Err
}

func (b *Base) CtxDone() <-chan struct{} {
	return b.Ctx.Done()
}

func (b *Base) SetCtx(ctx context.Context) {
	b.Ctx = ctx
}

func (b *Base) SetCancelFunc(cancelFunc context.CancelFunc) {
	b.CancelFunc = cancelFunc
}

func (b *Base) GetRetry() int {
	return b.Retry
}

func (b *Base) SetRetry(retry int) {
	b.Retry = retry
}

func (b *Base) Cancel() {
	b.SetStatus(StatusCanceling)
	b.CancelFunc()
}

var _ TaskBase = (*Base)(nil)
