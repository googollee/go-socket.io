package polling

import (
	"github.com/googollee/go-socket.io/engineio/frame"
)

type packet struct {
	Type frame.Type
	Body string
}
