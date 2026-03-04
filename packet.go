package main

var packetQueue = make(chan []byte, 500000)
