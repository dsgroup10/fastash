package main

import "time"

type RateLimiter struct {
	window time.Duration
	state  []time.Time
	index  int
}

func NewRateLimiter(window time.Duration, requests int) *RateLimiter {
	return &RateLimiter{
		window: window,
		state:  make([]time.Time, requests),
		index:  0,
	}
}

func (r *RateLimiter) ReadyAt() time.Time {
	return r.state[r.index]
}

func (r *RateLimiter) Update() {
	r.state[r.index] = time.Now().Add(r.window)
	r.index = (r.index + 1) % len(r.state)
}
