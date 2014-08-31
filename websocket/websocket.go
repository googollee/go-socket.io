package websocket

import (
	"github.com/googollee/go-engine.io/transport"
)

var Creater = transport.Creater{
	Name:   "websocket",
	Server: NewServer,
	Client: NewClient,
}
