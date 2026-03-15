package main

import (
	"log"
	"net"
	"sync"
	"time"
)

type Backend struct {
	Addr  string
	Conns []net.Conn
	mu    sync.Mutex
	next  int
}

type BackendPool struct {
	backends []Backend
}

func NewBackendPool(addrs []string, sockets int) *BackendPool {

	var backends []Backend

	for _, a := range addrs {

		var conns []net.Conn

		for i := 0; i < sockets; i++ {

			conn, err := net.Dial("tcp", a)
			if err != nil {
				log.Fatalf("backend connect failed %s", a)
			}

			if tcp, ok := conn.(*net.TCPConn); ok {
				tcp.SetNoDelay(true)
			}

			conns = append(conns, conn)
		}

		log.Println("connected backend:", a)

		backends = append(backends, Backend{
			Addr:  a,
			Conns: conns,
		})
	}

	return &BackendPool{
		backends: backends,
	}
}

func (b *Backend) Write(data []byte) error {

	b.mu.Lock()

	conn := b.Conns[b.next]
	idx := b.next
	b.next = (b.next + 1) % len(b.Conns)

	b.mu.Unlock()

	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))

	_, err := conn.Write(data)

	if err != nil {

		conn.Close()

		newConn, err2 := net.Dial("tcp", b.Addr)
		if err2 == nil {

			if tcp, ok := newConn.(*net.TCPConn); ok {
				tcp.SetNoDelay(true)
			}

			b.mu.Lock()
			b.Conns[idx] = newConn
			b.mu.Unlock()
		}
	}

	return err
}

func (p *BackendPool) Get(idx int) *Backend {
	return &p.backends[idx]
}
