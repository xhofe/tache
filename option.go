package tache

import "time"

type Options struct {
	Works           int
	Retry           int
	Timeout         *time.Duration
	PersistPath     string
	PersistDebounce *time.Duration
}

func DefaultOptions() *Options {
	return &Options{
		Works: 5,
		//Retry: 1,
	}
}

type Option func(*Options)

func WithOptions(opts Options) Option {
	return func(o *Options) {
		*o = opts
	}
}

func WithWorks(works int) Option {
	return func(o *Options) {
		o.Works = works
	}
}

func WithRetry(retry int) Option {
	return func(o *Options) {
		o.Retry = retry
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.Timeout = &timeout
	}
}

func WithPersistPath(path string) Option {
	return func(o *Options) {
		o.PersistPath = path
	}
}

func WithPersistDebounce(debounce time.Duration) Option {
	return func(o *Options) {
		o.PersistDebounce = &debounce
	}
}
