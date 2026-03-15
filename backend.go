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
}

func NewBackendPool(addrs []string) *BackendPool {

	var backends []Backend

	for _, a := range addrs {

		conn, err := net.Dial("tcp", a)
		if err != nil {
			log.Fatalf("backend connect failed %s", a)
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

	if err != nil {

		b.Conn.Close()

		conn, err2 := net.Dial("tcp", b.Addr)
		if err2 != nil {
			return err
		}

		b.Conn = conn
		_, err = b.Conn.Write(data)
	}

	return err
}

func (p *BackendPool) Get(idx int) *Backend {
	return &p.backends[idx]
}
