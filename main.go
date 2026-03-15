package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {

	config := LoadConfig()

	router := NewRouter(config.Backends, config.BackendSockets)

	workers := runtime.NumCPU() * 4

	for i := 0; i < workers; i++ {
		go StartWorker(router)
	}

	log.Println("TCAP Router started")

	go StartListener(config.ListenAddr)

	waitForShutdown()
}

func waitForShutdown() {

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-ctx.Done()

	stop()

	log.Println("shutdown signal received")
}
