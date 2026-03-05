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

	length := int(uint32(data[4])<<24 |
		uint32(data[5])<<16 |
		uint32(data[6])<<8 |
		uint32(data[7]))

	if length > len(data) {
		return M3UAMessage{}, false
	}

	msg := M3UAMessage{
		Class: data[2],
		Type:  data[3],
	}

	msg.Payload = data[8:length]

	return msg, true
}
