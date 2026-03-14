package main

import (
	"log"
	"net"
)

type Backend struct {
	Addr  string
	Conn  net.Conn
	Queue chan Packet // Async writer queue
}

type BackendPool struct {
	backends []Backend
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

		b := Backend{
			Addr:  a,
			Conn:  conn,
			Queue: make(chan Packet, 100000), // Large backlog cushion
		}
		
		backends = append(backends, b)

		// Start dedicated lock-free socket writer
		go startBackendWriter(b)
	}

	return &BackendPool{
		backends: backends,
	}
}

func startBackendWriter(b Backend) {
	for pkt := range b.Queue {
		// Write to TCP socket inline, lockless
		b.Conn.Write(pkt.Data)
		// Return safely to pool *after* successful write
		bufferPool.Put(pkt.Buffer)
	}
}

func (b *Backend) Write(pkt Packet) error {
	// Pushing pointer struct is non-blocking and zero-alloc
	b.Queue <- pkt
	return nil
}

func (p *BackendPool) Get(idx int) *Backend {
	return &p.backends[idx]
}
