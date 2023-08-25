package tache

import "time"

type Options struct {
	Max     int
	Retry   int
	Timeout *time.Duration
}

func DefaultOptions() *Options {
	return &Options{
		Max: 5,
	}
}

type Option func(*Options)

func WithOptions(opts Options) Option {
	return func(o *Options) {
		*o = opts
	}
}

func WithMax(max int) Option {
	return func(o *Options) {
		o.Max = max
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
