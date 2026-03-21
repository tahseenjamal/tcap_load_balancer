package main

import (
	"sync"
	"time"
)

const (
	shardCount = 256
	txTTL      = 60 * time.Second
)

///////////////////////////////////////////////////////////
// GLOBAL M3UA CONNECTION POOL
///////////////////////////////////////////////////////////

var (
	// STP side connections (Go → osmo-stp)
	m3uaPool []*M3UAConn

	// Backend connections (backend apps → Go)
	backendPool   []*M3UAConn
	backendPoolMu sync.RWMutex
)

///////////////////////////////////////////////////////////
// TRANSACTION STATE
///////////////////////////////////////////////////////////

type TxEntry struct {
	src int
	dst int
	ts  time.Time
}

type TxShard struct {
	mu sync.RWMutex
	m  map[uint64]TxEntry
}

type Router struct {
	shards [shardCount]TxShard
}

///////////////////////////////////////////////////////////
// INIT
///////////////////////////////////////////////////////////

func NewRouter() *Router {
	r := &Router{}

	for i := range r.shards {
		r.shards[i].m = make(map[uint64]TxEntry)
	}

	go r.cleanup()

	return r
}

func (r *Router) shard(id uint64) *TxShard {
	return &r.shards[id%shardCount]
}

///////////////////////////////////////////////////////////
// HASH (better than simple modulo)
///////////////////////////////////////////////////////////

func hash(id uint64, n int) int {
	return int((id ^ (id >> 32)) % uint64(n))
}

///////////////////////////////////////////////////////////
// ROUTING
///////////////////////////////////////////////////////////

func (r *Router) Route(msg TCAPMessage, pkt Packet) {
	switch msg.Type {

	///////////////////////////////////////////////////////
	// BEGIN
	///////////////////////////////////////////////////////

	case TCAP_BEGIN:

		if pkt.FromBackend {

			// Backend → STP
			if len(m3uaPool) == 0 {
				return
			}
			dst := hash(msg.OTID, len(m3uaPool))

			sh := r.shard(msg.OTID)

			sh.mu.Lock()
			sh.m[msg.OTID] = TxEntry{
				dst: dst,
				ts:  time.Now(),
			}
			sh.mu.Unlock()

			sendM3UA(dst, pkt.Data)

		} else {
			// STP → Backend (rare case)
			sendBackend(pkt.Data, pkt.Src)
		}

	///////////////////////////////////////////////////////
	// CONTINUE / END / ABORT
	///////////////////////////////////////////////////////

	case TCAP_CONTINUE, TCAP_END, TCAP_ABORT:

		sh := r.shard(msg.DTID)

		sh.mu.RLock()
		entry, ok := sh.m[msg.DTID]
		sh.mu.RUnlock()

		if !ok {
			return
		}

		if pkt.FromBackend {
			// Backend → STP
			sendM3UA(entry.dst, pkt.Data)
		} else {
			// STP → Backend
			sendBackend(pkt.Data, pkt.Src)
		}

		if msg.Type == TCAP_END || msg.Type == TCAP_ABORT {
			sh.mu.Lock()
			delete(sh.m, msg.DTID)
			sh.mu.Unlock()
		}
	}
}

///////////////////////////////////////////////////////////
// M3UA SEND (CRITICAL FIX)
///////////////////////////////////////////////////////////

func sendM3UA(idx int, data []byte) {
	if len(m3uaPool) == 0 {
		return
	}

	conn := m3uaPool[idx%len(m3uaPool)]

	if conn == nil {
		return
	}

	conn.SendData(data)
}

///////////////////////////////////////////////////////////
// CLEANUP
///////////////////////////////////////////////////////////

func (r *Router) cleanup() {
	ticker := time.NewTicker(30 * time.Second)

	for range ticker.C {

		now := time.Now()

		for i := range r.shards {

			sh := &r.shards[i]

			sh.mu.Lock()

			for k, v := range sh.m {
				if now.Sub(v.ts) > txTTL {
					delete(sh.m, k)
				}
			}

			sh.mu.Unlock()
		}
	}
}

func sendBackend(data []byte, idx int) {
	backendPoolMu.RLock()
	defer backendPoolMu.RUnlock()

	n := len(backendPool)
	if n == 0 {
		return
	}

	for i := 0; i < n; i++ {
		conn := backendPool[(idx+i)%n]

		if conn != nil && conn.active.Load() {
			conn.SendData(data)
			return
		}
	}
}
