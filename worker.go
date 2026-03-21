package main

func StartWorker(router *Router, q chan Packet) {
	for pkt := range q {

		sccp, ok := ParseSCCP(pkt.Data)
		if !ok {
			continue
		}

		tcap, ok := ParseTCAPASN1(sccp.Data)
		if !ok {
			continue
		}

		router.Route(tcap, pkt)
	}
}
