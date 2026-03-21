package main

type Packet struct {
	Data        []byte
	Src         int // connection index
	FromBackend bool
}
