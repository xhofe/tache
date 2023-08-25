package tache

import "runtime"

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
