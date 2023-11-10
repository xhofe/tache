package tache

// Status is the status of a task
type Status int

const (
	// StatusPending is the status of a task when it is pending
	StatusPending = iota
	// StatusRunning is the status of a task when it is running
	StatusRunning
	// StatusSucceeded is the status of a task when it succeeded
	StatusSucceeded
	// StatusCanceling is the status of a task when it is canceling
	StatusCanceling
	// StatusCanceled is the status of a task when it is canceled
	StatusCanceled
	// StatusErrored is the status of a task when it is errored (it will be retried)
	StatusErrored
	// StatusFailing is the status of a task when it is failing (executed OnFailed hook)
	StatusFailing
	// StatusFailed is the status of a task when it failed (no retry times left)
	StatusFailed
	// StatusWaitingRetry is the status of a task when it is waiting for retry
	StatusWaitingRetry
	// StatusBeforeRetry is the status of a task when it is executing OnBeforeRetry hook
	StatusBeforeRetry
)
