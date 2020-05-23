package main

import (
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

type Proxy struct {
	dialer      proxy.ContextDialer
	client      *http.Client
	rateLimiter *RateLimiter
}

func NewProxy(dialer proxy.ContextDialer, rateLimitWindow time.Duration, rateLimit int) *Proxy {
	return &Proxy{
		dialer: dialer,
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: dialer.DialContext,
				// default parameters from net/http
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		rateLimiter: NewRateLimiter(rateLimitWindow, rateLimit),
	}
}
