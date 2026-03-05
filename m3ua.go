package main

type M3UAMessage struct {
	Class   uint8
	Type    uint8
	Payload []byte
}

func ParseM3UA(data []byte) (M3UAMessage, bool) {

	if len(data) < 8 {
		return M3UAMessage{}, false
	}

	msg := M3UAMessage{
		Class: data[2],
		Type:  data[3],
	}

	// skip header
	msg.Payload = data[8:]

	return msg, true
}
