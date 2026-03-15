package main

func StartWorker(router *Router) {

	for pkt := range packetQueue {

		tcap, ok := ParseTCAPASN1(pkt.Data)
		if ok {
			router.Route(tcap, pkt.Data)
		}
	}
}
