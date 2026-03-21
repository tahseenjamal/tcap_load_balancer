package main

import (
	"encoding/binary"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ishidawataru/sctp"
)

///////////////////////////////////////////////////////////
// STATE MACHINE
///////////////////////////////////////////////////////////

const (
	STATE_DOWN = iota
	STATE_UP
	STATE_ACTIVE
)

///////////////////////////////////////////////////////////
// STRUCT
///////////////////////////////////////////////////////////

type M3UAConn struct {
	addr string

	conn *sctp.SCTPConn

	active atomic.Bool
	state  int

	dispatch func(Packet)

	writeQ chan []byte

	mu sync.Mutex
}

///////////////////////////////////////////////////////////
// INIT
///////////////////////////////////////////////////////////

func NewM3UAConn(addr string, dispatch func(Packet)) *M3UAConn {
	m := &M3UAConn{
		addr:     addr,
		dispatch: dispatch,
		writeQ:   make(chan []byte, 100000),
		state:    STATE_DOWN,
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
			log.Println("Resolve error:", err)
			time.Sleep(time.Second)
			continue
		}

		conn, err := sctp.DialSCTP("sctp", nil, raddr)
		if err != nil {
			log.Println("Dial error:", err)
			time.Sleep(time.Second)
			continue
		}

		m.mu.Lock()
		m.conn = conn
		m.mu.Unlock()

		log.Println("SCTP connected")

		m.state = STATE_DOWN
		m.active.Store(false)

		///////////////////////////////////////////////////
		// START ASP HANDSHAKE
		///////////////////////////////////////////////////

		m.startASP()

		///////////////////////////////////////////////////
		// BLOCKING READ LOOP
		///////////////////////////////////////////////////

		m.readLoop()

		log.Println("SCTP disconnected, retrying...")
		time.Sleep(time.Second)
	}
}

///////////////////////////////////////////////////////////
// ASP START
///////////////////////////////////////////////////////////

func (m *M3UAConn) startASP() {
	log.Println("Sending ASPUP")
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

		if n < 8 {
			continue
		}

		m.handle(buf[:n])
	}
}

///////////////////////////////////////////////////////////
// MESSAGE HANDLER
///////////////////////////////////////////////////////////

func (m *M3UAConn) handle(b []byte) {

	class := b[2]
	typ := b[3]

	switch class {

	///////////////////////////////////////////////////
	// ASPSM
	///////////////////////////////////////////////////

	case 3:
		if typ == 4 { // ASPUP_ACK
			log.Println("ASPUP_ACK received")

			m.state = STATE_UP

			// ⚠️ IMPORTANT: small delay for STP stability
			time.Sleep(50 * time.Millisecond)

			log.Println("Sending ASPAC")
			m.sendSimple(4, 1) // ASPAC
		}

	///////////////////////////////////////////////////
	// ASPTM
	///////////////////////////////////////////////////

	case 4:
		if typ == 4 { // ASPAC_ACK
			log.Println("ASP ACTIVE")

			m.state = STATE_ACTIVE
			m.active.Store(true)
		}

	///////////////////////////////////////////////////
	// TRANSFER
	///////////////////////////////////////////////////

	case 1:
		if typ == 1 { // DATA

			if !m.active.Load() {
				return
			}

			sccp := extractSCCP(b)
			if sccp != nil {
				m.dispatch(Packet{
					Data:        sccp,
					FromBackend: false,
				})
			}
		}
	}
}

///////////////////////////////////////////////////////////
// SEND DATA
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
// WRITE LOOP (WITH PPID FIX)
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

///////////////////////////////////////////////////////////
// FLUSH (CRITICAL: PPID = M3UA)
///////////////////////////////////////////////////////////

func (m *M3UAConn) flush(batch [][]byte) {

	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()

	if conn == nil {
		return
	}

	info := &sctp.SndRcvInfo{
		PPID: 3, // 🔥 CRITICAL FIX (M3UA PPID)
	}

	for _, msg := range batch {

		_, err := conn.SCTPWrite(msg, info)
		if err != nil {
			log.Println("write error:", err)
			return
		}
	}
}

///////////////////////////////////////////////////////////
// SIMPLE CONTROL MSG
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
// BUILD M3UA DATA
///////////////////////////////////////////////////////////

func buildM3UAData(payload []byte) []byte {

	rc := []byte{
		0x00, 0x06,
		0x00, 0x08,
		0x00, 0x00, 0x00, 0x01,
	}

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
// EXTRACT SCCP
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
