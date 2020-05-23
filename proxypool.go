package main

import (
	"container/heap"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

type ProxyPool struct {
	Proxies ProxyHeap
}

func NewProxyPool(window time.Duration, requests int, dialers ...proxy.ContextDialer) *ProxyPool {
	proxies := make(ProxyHeap, len(dialers))
	for i, dialer := range dialers {
		proxies[i] = NewProxy(dialer, window, requests)
	}
	return &ProxyPool{
		Proxies: proxies,
	}
}

func (p *ProxyPool) Do(req *http.Request) *http.Response {
	now := time.Now()
	sleepDur := p.Proxies[0].rateLimiter.ReadyAt().Sub(now)
	if sleepDur > 0 {
		log.Printf("Rate limit: sleeping for %v", sleepDur)
		time.Sleep(sleepDur)
	}
	res, err := p.Proxies[0].client.Do(req)
	if err != nil {
		panic(err)
	}
	p.Proxies[0].rateLimiter.Update()
	heap.Fix(&p.Proxies, 0)
	return res
}
