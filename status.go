package tache

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

type Status int
