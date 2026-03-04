package main

type TCAPMessage struct {
	Type int
	OTID uint64
	DTID uint64
}

const (
	TCAP_BEGIN = iota
	TCAP_CONTINUE
	TCAP_END
	TCAP_ABORT
)

/*
Stub parser.

Real implementation must decode ASN.1 BER TCAP.
*/
func ParseTCAP(data []byte) TCAPMessage {
	if len(data) == 0 {
		return TCAPMessage{}
	}

	// fake logic for testing
	if data[0]%3 == 0 {
		return TCAPMessage{
			Type: TCAP_BEGIN,
			OTID: uint64(data[0]),
		}
	}

	if data[0]%3 == 1 {
		return TCAPMessage{
			Type: TCAP_CONTINUE,
			DTID: uint64(data[0]),
		}
	}

	return TCAPMessage{
		Type: TCAP_END,
		DTID: uint64(data[0]),
	}
}
