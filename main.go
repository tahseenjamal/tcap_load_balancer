package main

import (
	"log"
	"runtime"
)

func main() {

	///////////////////////////////////////////////////////////
	// INIT ROUTER
	///////////////////////////////////////////////////////////

	router := NewRouter()

	///////////////////////////////////////////////////////////
	// WORKER POOL (CPU-bound parsing)
	///////////////////////////////////////////////////////////

	workers := runtime.NumCPU() * 4
	queues := make([]chan Packet, workers)

	for i := 0; i < workers; i++ {
		queues[i] = make(chan Packet, 100000)
		go StartWorker(router, queues[i])
	}

	///////////////////////////////////////////////////////////
	// DISPATCH FUNCTION (lock-free fanout)
	///////////////////////////////////////////////////////////

	dispatch := func(pkt Packet) {

		// simple hash (fast path)
		idx := int(pkt.Data[0]) % workers

		select {
		case queues[idx] <- pkt:
		default:
			// drop under extreme pressure (protect system)
		}
	}

	log.Println("Starting M3UA Router")

	///////////////////////////////////////////////////////////
	// 1. START M3UA SERVER (Backend → Go Router)
	///////////////////////////////////////////////////////////

	// Backend applications will connect here
	NewM3UAServer("0.0.0.0:2906", dispatch)

	log.Println("M3UA Server started on 0.0.0.0:2906")

	///////////////////////////////////////////////////////////
	// 2. START M3UA CLIENT POOL (Go Router → osmo-stp)
	///////////////////////////////////////////////////////////

	stpAddr := "127.0.0.1:2905" // 🔴 CHANGE to real STP IP in production

	for i := 0; i < 4; i++ {

		conn := NewM3UAConn(stpAddr, dispatch)

		m3uaPool = append(m3uaPool, conn)
	}

	log.Println("Connected to STP:", stpAddr, "connections:", len(m3uaPool))

	///////////////////////////////////////////////////////////
	// BLOCK FOREVER
	///////////////////////////////////////////////////////////

	select {}
}
