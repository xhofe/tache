package tache

import "context"

type Base struct {
	Progress float64 `json:"progress"`
	Status   Status  `json:"status"`
	ID       int64   `json:"id"`
	Retry    int     `json:"retry"`
	err      error
	ctx      context.Context
	cancel   context.CancelFunc
}

func (b *Base) SetProgress(progress float64) {
	b.Progress = progress
}

func (b *Base) GetProgress() float64 {
	return b.Progress
}

func (b *Base) SetStatus(status Status) {
	b.Status = status
}

func (b *Base) GetStatus() Status {
	return b.Status
}

func (b *Base) GetID() int64 {
	return b.ID
}

func (b *Base) SetID(id int64) {
	b.ID = id
}

func (b *Base) SetErr(err error) {
	b.err = err
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

func (b *Base) GetRetry() int {
	return b.Retry
}

func (b *Base) SetRetry(retry int) {
	b.Retry = retry
}

func (b *Base) Cancel() {
	b.SetStatus(StatusCanceling)
	b.cancel()
}

func (b *Base) Ctx() context.Context {
	return b.ctx
}

var _ TaskBase = (*Base)(nil)
