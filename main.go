package main

import (
	"log"
	"runtime"
)

func main() {

	config := LoadConfig()

	router := NewRouter(config.Backends)

	workers := runtime.NumCPU() * 2

	for i := 0; i < workers; i++ {
		go StartWorker(router)
	}

	log.Println("TCAP Router started")

	StartListener(config.ListenAddr)
}
