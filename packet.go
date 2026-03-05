package main

type Packet struct {
	Data []byte
}

var packetQueue = make(chan Packet, 500000)
