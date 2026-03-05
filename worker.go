package main

func StartWorker(router *Router) {

	for pkt := range packetQueue {

		m3ua, ok := ParseM3UA(pkt.Data)
		if ok {
			sccp, ok := ParseSCCP(m3ua.Payload)
			if ok {
				tcap, ok := ParseTCAPASN1(sccp.Payload)
				if ok {
					router.Route(tcap, pkt.Data)
				}
			}
		}

		// return buffer to pool
		bufferPool.Put(pkt.Buffer)
	}
}
