package main

import (
	"encoding/binary"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ishidawataru/sctp"
)

type M3UAConn struct {
	addr string

	conn *sctp.SCTPConn

	active atomic.Bool

	dispatch func(Packet)

	writeQ chan []byte

	mu sync.Mutex
}

func NewM3UAConn(addr string, dispatch func(Packet)) *M3UAConn {
	m := &M3UAConn{
		addr:     addr,
		dispatch: dispatch,
		writeQ:   make(chan []byte, 100000),
	}

	go m.connectLoop()
	go m.writeLoop()

	return m
}

///////////////////////////////////////////////////////////
// CONNECTION LOOP
///////////////////////////////////////////////////////////

func (m *M3UAConn) connectLoop() {
	for {

		log.Println("Connecting SCTP:", m.addr)

		raddr, err := sctp.ResolveSCTPAddr("sctp", m.addr)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		conn, err := sctp.DialSCTP("sctp", nil, raddr)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		m.mu.Lock()
		m.conn = conn
		m.mu.Unlock()

		log.Println("SCTP connected")

		m.active.Store(false)

		m.startASP()

		m.readLoop()

		log.Println("SCTP disconnected, retrying...")
		time.Sleep(time.Second)
	}
}

///////////////////////////////////////////////////////////
// ASP STATE MACHINE
///////////////////////////////////////////////////////////

func (m *M3UAConn) startASP() {
	m.sendSimple(3, 1) // ASPUP
}

///////////////////////////////////////////////////////////
// READ LOOP
///////////////////////////////////////////////////////////

func (m *M3UAConn) readLoop() {
	buf := make([]byte, 65535)

	for {

		n, err := m.conn.Read(buf)
		if err != nil {
			return
		}

		m.handle(buf[:n])
	}
}

///////////////////////////////////////////////////////////
// MESSAGE HANDLER
///////////////////////////////////////////////////////////

func (m *M3UAConn) handle(b []byte) {
	if len(b) < 8 {
		return
	}

	class := b[2]
	typ := b[3]

	switch class {

	case 3: // ASPSM
		if typ == 4 { // ASPUP_ACK
			log.Println("ASPUP_ACK")
			m.sendSimple(4, 1) // ASPAC
		}

	case 4: // ASPTM
		if typ == 4 { // ASPAC_ACK
			log.Println("ASP ACTIVE")
			m.active.Store(true)
		}

	case 1: // TRANSFER
		if typ == 1 { // DATA

			sccp := extractSCCP(b)
			if sccp != nil {
				m.dispatch(Packet{Data: sccp, FromBackend: false})
			}
		}
	}
}

///////////////////////////////////////////////////////////
// SEND DATA (NON-BLOCKING)
///////////////////////////////////////////////////////////

func (m *M3UAConn) SendData(sccp []byte) {
	if !m.active.Load() {
		return
	}

	msg := buildM3UAData(sccp)

	select {
	case m.writeQ <- msg:
	default:
		// drop under pressure
	}
}

///////////////////////////////////////////////////////////
// WRITE LOOP (BATCHED)
///////////////////////////////////////////////////////////

func (m *M3UAConn) writeLoop() {
	batch := make([][]byte, 0, 128)

	ticker := time.NewTicker(1 * time.Millisecond)

	for {
		select {

		case msg := <-m.writeQ:

			batch = append(batch, msg)

			if len(batch) >= 64 {
				m.flush(batch)
				batch = batch[:0]
			}

		case <-ticker.C:

			if len(batch) > 0 {
				m.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

func (m *M3UAConn) flush(batch [][]byte) {
	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()

	if conn == nil {
		return
	}

	for _, msg := range batch {
		_, err := conn.Write(msg)
		if err != nil {
			log.Println("write error:", err)
			return
		}
	}
}

///////////////////////////////////////////////////////////
// SIMPLE CONTROL MESSAGES
///////////////////////////////////////////////////////////

func (m *M3UAConn) sendSimple(class, typ uint8) {
	buf := make([]byte, 8)

	buf[0] = 1
	buf[2] = class
	buf[3] = typ

	binary.BigEndian.PutUint32(buf[4:], 8)

	select {
	case m.writeQ <- buf:
	default:
	}
}

///////////////////////////////////////////////////////////
// M3UA BUILD (DATA)
///////////////////////////////////////////////////////////

func buildM3UAData(payload []byte) []byte {
	// Routing Context TLV
	rc := []byte{
		0x00, 0x06,
		0x00, 0x08,
		0x00, 0x00, 0x00, 0x01,
	}

	// Protocol Data TLV
	pdLen := uint16(len(payload) + 4)

	pd := make([]byte, 4)
	binary.BigEndian.PutUint16(pd[0:], 0x0210)
	binary.BigEndian.PutUint16(pd[2:], pdLen)

	pd = append(pd, payload...)

	totalLen := 8 + len(rc) + len(pd)

	buf := make([]byte, 8)

	buf[0] = 1
	buf[2] = 1
	buf[3] = 1

	binary.BigEndian.PutUint32(buf[4:], uint32(totalLen))

	buf = append(buf, rc...)
	buf = append(buf, pd...)

	return buf
}

///////////////////////////////////////////////////////////
// EXTRACT SCCP FROM M3UA (TLV PARSER)
///////////////////////////////////////////////////////////

func extractSCCP(b []byte) []byte {
	i := 8

	for i+4 <= len(b) {

		tag := binary.BigEndian.Uint16(b[i:])
		length := int(binary.BigEndian.Uint16(b[i+2:]))

		if tag == 0x0210 {
			return b[i+4 : i+length]
		}

		i += (length + 3) &^ 3
	}

	return nil
}
