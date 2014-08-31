package polling

import (
	"github.com/googollee/go-engine.io/transport"
)

var XHRCreater = transport.ServerCreater{
	Name:   "polling",
	Server: nil,
	Client: nil,
}
