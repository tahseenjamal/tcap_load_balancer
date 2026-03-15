package main

import (
	"log"
	"net"
)

const (
	readBufferSize = 4 * 1024 * 1024 // 4MB socket buffer
	maxPacketSize  = 65535
)

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

	// Tune socket buffers for high throughput
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetReadBuffer(readBufferSize)
	}

	buf := make([]byte, maxPacketSize)

	for {

		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		// Allocate exact-size packet
		data := make([]byte, n)
		copy(data, buf[:n])

		packet := Packet{
			Data: data,
		}

		select {

		case packetQueue <- packet:

		default:
			// queue full, drop packet
			log.Println("packet queue full, dropping packet")
		}
	}
}
