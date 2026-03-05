package main

type Packet struct {
	Data   []byte
	Buffer *[]byte
}

var packetQueue = make(chan Packet, 500000)
