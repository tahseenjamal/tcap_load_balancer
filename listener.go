package main

import (
	"log"
	"net"
	"sync/atomic"
)

const (
	readBufferSize = 4 * 1024 * 1024 // 4MB socket buffer
	maxPacketSize  = 65535
)

var dropped uint64

func StartListener(addr string) {

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening on", addr)

	for {

		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {

	defer conn.Close()

	// TCP tuning for high throughput
	if tcp, ok := conn.(*net.TCPConn); ok {

		tcp.SetReadBuffer(readBufferSize)
		tcp.SetNoDelay(true)
	}

	buf := make([]byte, maxPacketSize)

	for {

		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		// allocate exact-sized packet
		data := make([]byte, n)
		copy(data, buf[:n])

		packet := Packet{
			Data: data,
		}

		select {

		case packetQueue <- packet:

		default:

			count := atomic.AddUint64(&dropped, 1)

			// throttle logging
			if count%1000 == 0 {
				log.Println("packet queue full, dropped:", count)
			}
		}
	}
}
