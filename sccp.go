package main

type SCCPMessage struct {
	Type    uint8
	Payload []byte
}

func ParseSCCP(data []byte) (SCCPMessage, bool) {

	if len(data) < 5 {
		return SCCPMessage{}, false
	}

	msg := SCCPMessage{
		Type: data[0],
	}

	// simplified: payload after SCCP header
	msg.Payload = data[5:]

	return msg, true
}
