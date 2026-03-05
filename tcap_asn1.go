package main

func ParseTCAPASN1(data []byte) (TCAPMessage, bool) {

	if len(data) < 4 {
		return TCAPMessage{}, false
	}

	tag := data[0]

	msg := TCAPMessage{}

	switch tag {

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

	// simple scan for OTID/DTID tags
	for i := 0; i < len(data)-2; i++ {

		if data[i] == 0x48 {
			msg.OTID = uint64(data[i+2])
		}

		if data[i] == 0x49 {
			msg.DTID = uint64(data[i+2])
		}
	}

	return msg, true
}
