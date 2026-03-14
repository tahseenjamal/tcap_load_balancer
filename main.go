package main

import (
	"log"
	"runtime"
)

func main() {
	config := LoadConfig()

	router := NewRouter(config.Backends)

	workerCount := runtime.NumCPU()

	workerQueues = make([]chan Packet, workerCount)

	for i := 0; i < workerCount; i++ {
		workerQueues[i] = make(chan Packet, 100000)
		go StartWorker(router, workerQueues[i])
	}

	log.Println("TCAP Load Balancer Started")

	StartListener(config.ListenAddr)
}
