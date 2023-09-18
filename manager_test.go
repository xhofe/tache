package tache_test

import (
	"testing"

	"github.com/xhofe/tache"
)

type TestTask struct {
	tache.Base
	Data string
	do   func(*TestTask) error
}

func (t *TestTask) Run() error {
	return t.do(t)
}

func TestManager_Add(t *testing.T) {
	tm := tache.NewManager[*TestTask]()
	task := &TestTask{}
	tm.Add(task)
	t.Logf("%+v", task)
}

func TestWithRetry(t *testing.T) {
	tm := tache.NewManager[*TestTask](tache.WithRetry(3))
	var i int
	task := &TestTask{
		do: func(task *TestTask) error {
			i++
			if i < 4 {
				return tache.NewErr("test")
			}
			return nil
		},
	}
	tm.Add(task)
	tm.Wait()
	if task.GetRetry() != 3 || task.GetStatus() != tache.StatusSucceeded {
		t.Errorf("retry error, retry: %d,status: %d", task.GetRetry(), task.GetStatus())
	} else {
		t.Logf("retry success, retry: %d,status: %d", task.GetRetry(), task.GetStatus())
	}
}

func TestWithPersistPath(t *testing.T) {
	tm := tache.NewManager[*TestTask](tache.WithPersistPath("./test.json"))
	task := &TestTask{
		do: func(task *TestTask) error {
			return nil
		},
		Data: "haha",
	}
	tm.Add(task)
	tm.Wait()
	t.Logf("%+v", task)
}
