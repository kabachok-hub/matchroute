package wnet

import "strings"

const AdapterName = "MatchRoute"

func IsAppAdapter(name string) bool {
	name = strings.ToLower(name)
	return strings.HasPrefix(name, strings.ToLower(AdapterName)) || strings.HasPrefix(name, "fnport")
}

// Dev: a packet device, the Wintun adapter or a stand-in in tests.
// ReadPacket is called from one goroutine; WritePacket may be called from many.
type Dev interface {
	ReadPacket(buf []byte) (int, error)
	WritePacket(pkt []byte) error
	Close() error
}
