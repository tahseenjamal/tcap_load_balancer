package main

import (
	"log"
	"net"
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
			continue
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 65535)

	for {

		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		select {
		case packetQueue <- data:
		default:
			log.Println("packet queue full, dropping packet")
		}
	}
}
