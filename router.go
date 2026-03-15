package main

import (
	"hash/fnv"
	"sync"
	"time"
)

const shardCount = 256
const txTTL = 60 * time.Second

type TxEntry struct {
	backend int
	ts      time.Time
}

type TxShard struct {
	mu sync.RWMutex
	m  map[uint64]TxEntry
}

type Router struct {
	pool   *BackendPool
	shards [shardCount]TxShard
}

func NewRouter(backends []string, sockets int) *Router {

	r := &Router{
		pool: NewBackendPool(backends, sockets),
	}

	for i := range r.shards {
		r.shards[i].m = make(map[uint64]TxEntry)
	}

	go r.cleanup()

	return r
}

func (r *Router) shard(id uint64) *TxShard {
	return &r.shards[id%shardCount]
}

func (r *Router) Route(msg TCAPMessage, raw []byte) {

	switch msg.Type {

	case TCAP_BEGIN:

		idx := hashBackend(msg.OTID, len(r.pool.backends))

		shard := r.shard(msg.OTID)

		shard.mu.Lock()
		shard.m[msg.OTID] = TxEntry{backend: idx, ts: time.Now()}
		shard.mu.Unlock()

		r.pool.Get(idx).Write(raw)

	case TCAP_CONTINUE:

		shard := r.shard(msg.DTID)

		shard.mu.RLock()
		entry, ok := shard.m[msg.DTID]
		shard.mu.RUnlock()

		var idx int

		if ok {
			idx = entry.backend
		} else {
			idx = hashBackend(msg.DTID, len(r.pool.backends))
		}

		if msg.OTID != 0 {

			shard2 := r.shard(msg.OTID)

			shard2.mu.Lock()
			shard2.m[msg.OTID] = TxEntry{backend: idx, ts: time.Now()}
			shard2.mu.Unlock()
		}

		r.pool.Get(idx).Write(raw)

	case TCAP_END, TCAP_ABORT:

		shard := r.shard(msg.DTID)

		shard.mu.RLock()
		entry, ok := shard.m[msg.DTID]
		shard.mu.RUnlock()

		var idx int

		if ok {
			idx = entry.backend
		} else {
			idx = hashBackend(msg.DTID, len(r.pool.backends))
		}

		r.pool.Get(idx).Write(raw)

		shard.mu.Lock()
		delete(shard.m, msg.DTID)
		shard.mu.Unlock()
	}
}

func (r *Router) cleanup() {

	ticker := time.NewTicker(30 * time.Second)

	for range ticker.C {

		now := time.Now()

		for i := range r.shards {

			shard := &r.shards[i]

			shard.mu.Lock()

			for k, v := range shard.m {

				if now.Sub(v.ts) > txTTL {
					delete(shard.m, k)
				}
			}

			shard.mu.Unlock()
		}
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
