package main

type ProxyHeap []*Proxy

func (h ProxyHeap) Len() int {
	return len(h)
}

func (h ProxyHeap) Less(i, j int) bool {
	return h[i].rateLimiter.ReadyAt().Before(h[j].rateLimiter.ReadyAt())
}

func (h ProxyHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *ProxyHeap) Push(x interface{}) {
	*h = append(*h, x.(*Proxy))
}

func (h *ProxyHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*h = old[0 : n-1]
	return x
}
