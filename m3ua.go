package main

import "encoding/binary"

const (
	M3UA_VERSION = 1
	M3UA_HEADER  = 8
)

type M3UAMessage struct {
	Version uint8
	Class   uint8
	Type    uint8
	Length  uint32
	Payload []byte
}

func ParseM3UA(data []byte) (M3UAMessage, bool) {

	// Minimum header length
	if len(data) < M3UA_HEADER {
		return M3UAMessage{}, false
	}

	version := data[0]

	// Validate version
	if version != M3UA_VERSION {
		return M3UAMessage{}, false
	}

	msgClass := data[2]
	msgType := data[3]

	length := binary.BigEndian.Uint32(data[4:8])

	// Length validation
	if length < M3UA_HEADER || int(length) > len(data) {
		return M3UAMessage{}, false
	}

	msg := M3UAMessage{
		Version: version,
		Class:   msgClass,
		Type:    msgType,
		Length:  length,
		Payload: data[M3UA_HEADER:length],
	}

	return msg, true
}
