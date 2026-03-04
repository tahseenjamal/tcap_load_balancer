package main

import (
	"log"
)

type Router struct {
	txTable *TxTable
	pool    *BackendPool
}

func NewRouter(backends []string) *Router {
	return &Router{
		txTable: NewTxTable(),
		pool:    NewBackendPool(backends),
	}
}

func (r *Router) Route(msg TCAPMessage, raw []byte) {
	switch msg.Type {

	case TCAP_BEGIN:

		backend, idx := r.pool.Next()

		r.txTable.Store(msg.OTID, idx)

		_, err := backend.Conn.Write(raw)
		if err != nil {
			log.Println("backend write error:", err)
		}

	case TCAP_CONTINUE:

		tx, ok := r.txTable.Lookup(msg.DTID)

		if ok {

			if msg.OTID != 0 {
				r.txTable.Store(msg.OTID, tx.Backend)
			}

			backend := r.pool.backends[tx.Backend]

			_, err := backend.Conn.Write(raw)
			if err != nil {
				log.Println("backend write error:", err)
			}
		}

	case TCAP_END, TCAP_ABORT:

		tx, ok := r.txTable.Lookup(msg.DTID)

		if ok {

			backend := r.pool.backends[tx.Backend]

			_, err := backend.Conn.Write(raw)
			if err != nil {
				log.Println("backend write error:", err)
			}

			r.txTable.Delete(msg.DTID)
		}
	}
}
