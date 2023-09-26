package tache

import "time"

// Options is the options for manager
type Options struct {
	Works           int
	Retry           int
	Timeout         *time.Duration
	PersistPath     string
	PersistDebounce *time.Duration
}

// DefaultOptions returns default options
func DefaultOptions() *Options {
	persistDebounce := 3 * time.Second
	return &Options{
		Works: 5,
		//Retry: 1,
		PersistDebounce: &persistDebounce,
	}
}

// Option is the option for manager
type Option func(*Options)

// WithOptions set options
func WithOptions(opts Options) Option {
	return func(o *Options) {
		*o = opts
	}
}

// WithWorks set works
func WithWorks(works int) Option {
	return func(o *Options) {
		o.Works = works
	}
}

// WithRetry set retry
func WithRetry(retry int) Option {
	return func(o *Options) {
		o.Retry = retry
	}
}

// WithTimeout set timeout
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.Timeout = &timeout
	}
}

// WithPersistPath set persist path
func WithPersistPath(path string) Option {
	return func(o *Options) {
		o.PersistPath = path
	}
}

// WithPersistDebounce set persist debounce
func WithPersistDebounce(debounce time.Duration) Option {
	return func(o *Options) {
		o.PersistDebounce = &debounce
	}
}
