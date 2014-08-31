package polling

import (
	"github.com/googollee/go-engine.io/transport"
)

var XHRCreater = transport.Creater{
	Name:   "polling",
	Server: nil,
	Client: nil,
}
