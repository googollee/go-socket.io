package engineio

import (
	"io"
)

type Handler interface {
	OnOpen(socket Socket)
	OnMessage(socket Socket, r io.Reader)
	OnClose(socket Socket)
	OnError(socket Socket, err error)
}
