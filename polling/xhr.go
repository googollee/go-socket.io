package polling

import (
	"github.com/googollee/go-engine.io/transport"
)

var Creater = transport.Creater{
	Name:   "polling",
	Server: NewServer,
	Client: nil,
}
