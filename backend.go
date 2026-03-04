package main

import (
	"log"
	"net"
	"sync/atomic"
)

type Backend struct {
	Addr string
	Conn net.Conn
}

type BackendPool struct {
	backends []Backend
	counter  uint64
}

func NewBackendPool(addrs []string) *BackendPool {
	var backends []Backend

	for _, a := range addrs {

		conn, err := net.Dial("tcp", a)
		if err != nil {
			log.Fatalf("failed to connect backend %s: %v", a, err)
		}

		backends = append(backends, Backend{
			Addr: a,
			Conn: conn,
		})
	}

	return &BackendPool{backends: backends}
}

func (p *BackendPool) Next() (Backend, int) {
	i := atomic.AddUint64(&p.counter, 1)
	idx := int(i) % len(p.backends)

	return p.backends[idx], idx
}
