package main

func StartWorker(router *Router) {

	for pkt := range packetQueue {

		m3ua, ok := ParseM3UA(pkt.Data)
		if !ok {
			continue
		}

		sccp, ok := ParseSCCP(m3ua.Payload)
		if !ok {
			continue
		}

		tcap, ok := ParseTCAPASN1(sccp.Payload)
		if !ok {
			continue
		}

		router.Route(tcap, pkt.Data)
	}
}
