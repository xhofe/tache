package tache

import (
	"runtime"
	"sync"
	"time"
)

func sliceContains[T comparable](slice []T, v T) bool {
	for _, vv := range slice {
		if vv == v {
			return true
		}
	}
	return false
}

func getCurrentGoroutineStack() string {
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func newDebounce(f func(), interval time.Duration) func() {
	var timer *time.Timer
	var lock sync.Mutex
	return func() {
		lock.Lock()
		defer lock.Unlock()
		if timer == nil {
			timer = time.AfterFunc(interval, f)
		} else {
			timer.Reset(interval)
		}
	}
}
