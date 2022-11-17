package packet

import (
	"github.com/vchitai/go-socket.io/v4/engineio/frame"
)

type Frame struct {
	FType frame.Type
	Data  []byte
}

type Packet struct {
	FType frame.Type
	PType Type
	Data  []byte
}
