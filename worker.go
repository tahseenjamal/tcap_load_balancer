package main

func StartWorker(router *Router) {
	for pkt := range packetQueue {

		msg := ParseTCAP(pkt)

		router.Route(msg, pkt)
	}
}
