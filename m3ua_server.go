package main

import (
	"log"

	"github.com/ishidawataru/sctp"
)

type M3UAServer struct {
	addr     string
	dispatch func(Packet)
}

func NewM3UAServer(addr string, dispatch func(Packet)) {
	s := &M3UAServer{
		addr:     addr,
		dispatch: dispatch,
	}

	go s.listen()
}

func (s *M3UAServer) listen() {
	laddr, err := sctp.ResolveSCTPAddr("sctp", s.addr)
	if err != nil {
		log.Fatal("Resolve error:", err)
	}

	ln, err := sctp.ListenSCTP("sctp", laddr)
	if err != nil {
		log.Fatal("Listen error:", err)
	}

	log.Println("M3UA Server listening on", s.addr)

	for {
		conn, err := ln.AcceptSCTP()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}

		log.Println("Backend connected:", conn.RemoteAddr())

		m := &M3UAConn{
			conn:     conn,
			dispatch: s.dispatch,
			writeQ:   make(chan []byte, 100000),
		}

		backendPoolMu.Lock()

		// 🔥 reuse empty slot
		idx := -1
		for i, c := range backendPool {
			if c == nil {
				backendPool[i] = m
				idx = i
				break
			}
		}

		// 🔥 if no empty slot → append
		if idx == -1 {
			backendPool = append(backendPool, m)
			idx = len(backendPool) - 1
		}

		backendPoolMu.Unlock()

		log.Println("Backend index assigned:", idx)

		go m.readLoopServer(idx)
		go m.writeLoop()
	}
}

func (m *M3UAConn) readLoopServer(idx int) {
	buf := make([]byte, 65535)

	active := false

	for {
		n, err := m.conn.Read(buf)
		if err != nil {
			log.Println("Backend disconnected:", idx)

			// 🔥 FIX: REMOVE FROM POOL
			backendPoolMu.Lock()
			if idx < len(backendPool) {
				backendPool[idx] = nil
			}
			backendPoolMu.Unlock()

			m.active.Store(false)
			return
		}

		if n < 8 {
			continue
		}

		b := buf[:n]

		class := b[2]
		typ := b[3]

		switch class {

		case 3: // ASPSM
			if typ == 1 {
				m.sendSimple(3, 4)
			}

		case 4: // ASPTM
			if typ == 1 {
				m.sendSimple(4, 4)
				active = true
				m.active.Store(true)
				log.Println("Backend ASP ACTIVE:", idx)
			}

		case 1: // TRANSFER
			if typ == 1 && active {

				sccp := extractSCCP(b)

				if sccp != nil {
					m.dispatch(Packet{
						Data:        sccp,
						Src:         idx,
						FromBackend: true,
					})
				}
			}
		}
	}
}
