package main

import (
	"log"
	"runtime"
)

func main() {
	config := LoadConfig()

	router := NewRouter(config.Backends)

	workerCount := runtime.NumCPU()

	for i := 0; i < workerCount; i++ {
		go StartWorker(router)
	}

	go router.StartCleanup()

	log.Println("TCAP Load Balancer Started")

	StartListener(config.ListenAddr)
}
