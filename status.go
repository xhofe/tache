package tache

// Status is the status of a task
type Status int

const (
	StatusPending = iota
	StatusRunning
	StatusSucceeded
	StatusCanceling
	StatusCanceled
	StatusErrored
	StatusFailing
	// StatusFailed is the status of a task when it failed (no retry times left)
	StatusFailed
	StatusWaitingRetry
	StatusBeforeRetry
)
