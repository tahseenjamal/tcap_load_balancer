package main

/*
Minimal TCAP ASN.1 parser used only for routing.

Extracts:
- Message Type
- OTID
- DTID
*/

func ParseTCAPASN1(data []byte) (TCAPMessage, bool) {

	if len(data) < 4 {
		return TCAPMessage{}, false
	}

	msg := TCAPMessage{}

	switch data[0] {

	case 0x62:
		msg.Type = TCAP_BEGIN

	case 0x65:
		msg.Type = TCAP_CONTINUE

	case 0x64:
		msg.Type = TCAP_END

	case 0x67:
		msg.Type = TCAP_ABORT

	default:
		return TCAPMessage{}, false
	}

	for i := 0; i < len(data)-2; i++ {

		if msg.OTID != 0 && msg.DTID != 0 {
			break
		}

		switch data[i] {

		case 0x48:

			length := int(data[i+1])

			if i+2+length > len(data) {
				continue
			}

			var val uint64

			for j := 0; j < length; j++ {
				val = (val << 8) | uint64(data[i+2+j])
			}

			msg.OTID = val

		case 0x49:

			length := int(data[i+1])

			if i+2+length > len(data) {
				continue
			}

			var val uint64

			for j := 0; j < length; j++ {
				val = (val << 8) | uint64(data[i+2+j])
			}

			msg.DTID = val
		}
	}

	return msg, true
}
