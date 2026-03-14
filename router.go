package main

import (
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

func (r *Router) Route(msg TCAPMessage, pkt Packet) {

	switch msg.Type {

	case TCAP_BEGIN:

		if msg.OTID == 0 {
			log.Println("invalid TCAP BEGIN: missing OTID")
			// Must return to pool if we drop early
			bufferPool.Put(pkt.Buffer)
			return
		}

		idx := hashBackend(msg.OTID, len(r.pool.backends))
		backend := r.pool.Get(idx)

		if err := backend.Write(pkt); err != nil {
			log.Println("backend write error:", err)
		}

	case TCAP_CONTINUE:

		if msg.DTID == 0 {
			log.Println("invalid TCAP CONTINUE: missing DTID")
			bufferPool.Put(pkt.Buffer)
			return
		}

		idx := hashBackend(msg.DTID, len(r.pool.backends))
		backend := r.pool.Get(idx)

		if err := backend.Write(pkt); err != nil {
			log.Println("backend write error:", err)
		}

	case TCAP_END, TCAP_ABORT:

		if msg.DTID == 0 {
			log.Println("invalid TCAP END/ABORT: missing DTID")
			bufferPool.Put(pkt.Buffer)
			return
		}

		idx := hashBackend(msg.DTID, len(r.pool.backends))
		backend := r.pool.Get(idx)

		if err := backend.Write(pkt); err != nil {
			log.Println("backend write error:", err)
		}
	}
}

// inline zero-allocation hash
func hashBackend(otid uint64, count int) int {
	h := otid
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return int(h % uint64(count))
}
