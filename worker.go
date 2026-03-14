package main

func StartWorker(router *Router, queue chan Packet) {

	for pkt := range queue {

		handled := false
		m3ua, ok := ParseM3UA(pkt.Data)
		if ok {
			sccp, ok := ParseSCCP(m3ua.Payload)
			if ok {
				tcap, ok := ParseTCAPASN1(sccp.Payload)
				if ok {
					// Route takes ownership of pkt, will handle putting buffer back to pool
					router.Route(tcap, pkt)
					handled = true
				}
			}
		}

		if !handled {
			// Only return buffer if we failed to route it entirely 
			bufferPool.Put(pkt.Buffer)
		}
	}
}
