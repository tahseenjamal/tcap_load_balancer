package main

import (
	"hash/fnv"
	"log"
	"sync"
)

type Router struct {
	pool *BackendPool
	tx   map[uint64]int
	mu   sync.RWMutex
}

func NewRouter(backends []string) *Router {

	return &Router{
		pool: NewBackendPool(backends),
		tx:   make(map[uint64]int),
	}
}

func (r *Router) Route(msg TCAPMessage, raw []byte) {

	switch msg.Type {

	case TCAP_BEGIN:

		if msg.OTID == 0 {
			log.Println("invalid BEGIN")
			return
		}

		idx := hashBackend(msg.OTID, len(r.pool.backends))

		r.mu.Lock()
		r.tx[msg.OTID] = idx
		r.mu.Unlock()

		r.pool.Get(idx).Write(raw)

	case TCAP_CONTINUE:

		if msg.DTID == 0 {
			return
		}

		r.mu.RLock()
		idx, ok := r.tx[msg.DTID]
		r.mu.RUnlock()

		if !ok {
			idx = hashBackend(msg.DTID, len(r.pool.backends))
		}

		if msg.OTID != 0 {

			r.mu.Lock()
			r.tx[msg.OTID] = idx
			r.mu.Unlock()
		}

		r.pool.Get(idx).Write(raw)

	case TCAP_END, TCAP_ABORT:

		if msg.DTID == 0 {
			return
		}

		r.mu.RLock()
		idx, ok := r.tx[msg.DTID]
		r.mu.RUnlock()

		if !ok {
			idx = hashBackend(msg.DTID, len(r.pool.backends))
		}

		r.pool.Get(idx).Write(raw)

		r.mu.Lock()
		delete(r.tx, msg.DTID)
		r.mu.Unlock()
	}
}

func hashBackend(id uint64, count int) int {

	h := fnv.New32a()

	var b [8]byte

	for i := 0; i < 8; i++ {
		b[i] = byte(id >> (8 * i))
	}

	h.Write(b[:])

	return int(h.Sum32()) % count
}
