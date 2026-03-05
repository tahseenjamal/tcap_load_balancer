package main

type SCCPMessage struct {
	Type    uint8
	Payload []byte
}

func ParseSCCP(data []byte) (SCCPMessage, bool) {

	if len(data) < 4 {
		return SCCPMessage{}, false
	}

	msg := SCCPMessage{
		Type: data[0],
	}

	offset := int(data[1])

	if offset <= 0 || offset >= len(data) {
		return SCCPMessage{}, false
	}

	msg.Payload = data[offset:]

	return msg, true
}
