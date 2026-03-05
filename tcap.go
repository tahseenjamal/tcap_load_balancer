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
