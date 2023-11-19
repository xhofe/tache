package tache

import "context"

// Base is the base struct for all tasks to implement TaskBase interface
type Base struct {
	progress        float64
	status          Status
	id              string
	retry, maxRetry int
	err             error
	ctx             context.Context
	cancel          context.CancelFunc
	persist         func()
}

func (b *Base) SetProgress(progress float64) {
	b.progress = progress
	b.Persist()
}

func (b *Base) GetProgress() float64 {
	return b.progress
}

func (b *Base) SetStatus(status Status) {
	b.status = status
	b.Persist()
}

func (b *Base) GetStatus() Status {
	return b.status
}

func (b *Base) GetID() string {
	return b.id
}

func (b *Base) SetID(id string) {
	b.id = id
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
	return b.retry, b.maxRetry
}

func (b *Base) SetRetry(retry int, maxRetry int) {
	b.retry, b.maxRetry = retry, maxRetry
}

func (b *Base) Cancel() {
	b.SetStatus(StatusCanceling)
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

var _ TaskBase = (*Base)(nil)
