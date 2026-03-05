package main

func StartWorker(router *Router) {

	for pkt := range packetQueue {

		msg := ParseTCAP(pkt.Data)

		router.Route(msg, pkt.Data)
	}
}
