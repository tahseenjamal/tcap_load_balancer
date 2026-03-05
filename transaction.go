package main

import (
	"sync"
	"time"
)

const shardCount = 256

type TxEntry struct {
	Backend  int
	LastSeen int64
}

type Shard struct {
	sync.RWMutex
	table map[uint64]TxEntry
}

type TxTable struct {
	shards [shardCount]*Shard
}

func NewTxTable() *TxTable {
	t := &TxTable{}

	for i := 0; i < shardCount; i++ {
		t.shards[i] = &Shard{
			table: make(map[uint64]TxEntry),
		}
	}

	return t
}

func (t *TxTable) shard(id uint64) *Shard {
	return t.shards[id%shardCount]
}

func (t *TxTable) Store(id uint64, backend int) {
	s := t.shard(id)

	s.Lock()
	s.table[id] = TxEntry{backend, time.Now().Unix()}
	s.Unlock()
}

func (t *TxTable) Lookup(id uint64) (TxEntry, bool) {
	s := t.shard(id)

	s.Lock()
	v, ok := s.table[id]

	if ok {
		v.LastSeen = time.Now().Unix()
		s.table[id] = v
	}

	s.Unlock()

	return v, ok
}

func (t *TxTable) Delete(id uint64) {
	s := t.shard(id)

	s.Lock()
	delete(s.table, id)
	s.Unlock()
}

func (r *Router) StartCleanup() {
	for {

		time.Sleep(5 * time.Second)

		now := time.Now().Unix()

		for _, shard := range r.txTable.shards {

			shard.Lock()

			for k, v := range shard.table {
				if now-v.LastSeen > 60 {
					delete(shard.table, k)
				}
			}

			shard.Unlock()
		}
	}
}
