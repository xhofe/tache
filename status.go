package tache

// Status is the status of a task
type Status int

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
	StatusBeforeRetry
)
