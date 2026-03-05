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

		if msg.OTID == 0 {
			log.Println("invalid TCAP BEGIN: missing OTID")
			return
		}

		backend, idx := r.pool.Next()

		// store transaction → backend mapping
		r.txTable.Store(msg.OTID, idx)

		err := backend.Write(raw)
		if err != nil {
			log.Println("backend write error:", err)
		}

	case TCAP_CONTINUE:

		if msg.DTID == 0 {
			log.Println("invalid TCAP CONTINUE: missing DTID")
			return
		}

		tx, ok := r.txTable.Lookup(msg.DTID)

		if !ok {
			log.Println("transaction not found for DTID:", msg.DTID)
			return
		}

		backend := &r.pool.backends[tx.Backend]

		// update OTID mapping if new one appears
		if msg.OTID != 0 {
			r.txTable.Store(msg.OTID, tx.Backend)
		}

		err := backend.Write(raw)
		if err != nil {
			log.Println("backend write error:", err)
		}

	case TCAP_END, TCAP_ABORT:

		if msg.DTID == 0 {
			log.Println("invalid TCAP END/ABORT: missing DTID")
			return
		}

		tx, ok := r.txTable.Lookup(msg.DTID)

		if !ok {
			log.Println("transaction not found for DTID:", msg.DTID)
			return
		}

		backend := &r.pool.backends[tx.Backend]

		err := backend.Write(raw)
		if err != nil {
			log.Println("backend write error:", err)
		}

		// remove session after transaction completes
		r.txTable.Delete(msg.DTID)
	}
}
