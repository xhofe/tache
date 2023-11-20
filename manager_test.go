package tache_test

import (
	"github.com/xhofe/tache"
	"testing"
	"time"
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
	tm := tache.NewManager[*TestTask](tache.WithMaxRetry(3))
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
	retry, maxRetry := task.GetRetry()
	if retry != 3 || task.GetState() != tache.StateSucceeded {
		t.Errorf("retry error, retry: %d, maxRetry: %d, State: %d", retry, maxRetry, task.GetState())
	} else {
		t.Logf("retry success, retry: %d, maxRetry: %d, State: %d", retry, maxRetry, task.GetState())
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
	time.Sleep(4 * time.Second)
}
