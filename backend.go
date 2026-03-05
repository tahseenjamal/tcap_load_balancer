package main

import (
	"log"
	"net"
	"sync"
)

type Backend struct {
	Addr string
	Conn net.Conn
	mu   sync.Mutex
}

type BackendPool struct {
	backends []Backend
	counter  uint64
}

func NewBackendPool(addrs []string) *BackendPool {

	if len(addrs) == 0 {
		log.Fatal("no backends configured")
	}

	var backends []Backend

	for _, a := range addrs {

		conn, err := net.Dial("tcp", a)
		if err != nil {
			log.Fatalf("failed to connect backend %s: %v", a, err)
		}

		log.Println("connected backend:", a)

		backends = append(backends, Backend{
			Addr: a,
			Conn: conn,
		})
	}

	return &BackendPool{
		backends: backends,
	}
}

func (b *Backend) Write(data []byte) error {

	b.mu.Lock()
	defer b.mu.Unlock()

	_, err := b.Conn.Write(data)
	return err
}

func (p *BackendPool) Get(idx int) *Backend {
	return &p.backends[idx]
}
