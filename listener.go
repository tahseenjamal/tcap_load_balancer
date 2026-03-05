package main

import (
	"log"
	"net"
	"strconv"

	"github.com/ishidawataru/sctp"
)

func StartListener(addr string) {

	ip, port, _ := net.SplitHostPort(addr)

	sctpAddr := &sctp.SCTPAddr{
		IPAddrs: []net.IPAddr{
			{IP: net.ParseIP(ip)},
		},
		Port: atoi(port),
	}

	ln, err := sctp.ListenSCTP("sctp", sctpAddr)
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

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
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
		case packetQueue <- Packet{Data: data}:
		default:
			log.Println("packet queue full, dropping packet")
		}
	}
}
