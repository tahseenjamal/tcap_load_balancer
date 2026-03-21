package main

type SCCPMessage struct {
	Data []byte
}

func ParseSCCP(data []byte) (SCCPMessage, bool) {
	if len(data) < 5 {
		return SCCPMessage{}, false
	}

	ptr := int(data[3])
	if ptr >= len(data) {
		return SCCPMessage{}, false
	}

	return SCCPMessage{
		Data: data[ptr:],
	}, true
}
