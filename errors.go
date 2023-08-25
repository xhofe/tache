package tache

import "errors"

type TacheError struct {
	Msg string
}

func (e *TacheError) Error() string {
	return e.Msg
}

func NewErr(msg string) error {
	return &TacheError{Msg: msg}
}

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrTaskRunning  = errors.New("task is running")
)
