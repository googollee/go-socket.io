package packet

import (
	"github.com/googollee/go-socket.io/engineio/frame"
)

type Frame struct {
	t
	FType frame.Type
	Data  []byte
}

type Packet struct {
	FType frame.Type
	PType Type
	Data  []byte
}
