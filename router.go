package main

import (
	"hash/fnv"
	"log"
)

type Router struct {
	pool *BackendPool
}

func NewRouter(backends []string) *Router {
	return &Router{
		pool: NewBackendPool(backends),
	}
}

func (r *Router) Route(msg TCAPMessage, raw []byte) {

	switch msg.Type {

	case TCAP_BEGIN:

		if msg.OTID == 0 {
			log.Println("invalid TCAP BEGIN: missing OTID")
			return
		}

		idx := hashBackend(msg.OTID, len(r.pool.backends))
		backend := r.pool.Get(idx)

		if err := backend.Write(raw); err != nil {
			log.Println("backend write error:", err)
		}

	case TCAP_CONTINUE:

		if msg.DTID == 0 {
			log.Println("invalid TCAP CONTINUE: missing DTID")
			return
		}

		idx := hashBackend(msg.DTID, len(r.pool.backends))
		backend := r.pool.Get(idx)

		if err := backend.Write(raw); err != nil {
			log.Println("backend write error:", err)
		}

	case TCAP_END, TCAP_ABORT:

		if msg.DTID == 0 {
			log.Println("invalid TCAP END/ABORT: missing DTID")
			return
		}

		idx := hashBackend(msg.DTID, len(r.pool.backends))
		backend := r.pool.Get(idx)

		if err := backend.Write(raw); err != nil {
			log.Println("backend write error:", err)
		}
	}
}

func hashBackend(otid uint64, count int) int {
	h := fnv.New32a()
	var b [8]byte
	for i := 0; i < 8; i++ {
		b[i] = byte(otid >> (8 * i))
	}
	h.Write(b[:])
	return int(h.Sum32()) % count
}
