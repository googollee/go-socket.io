package transport

import (
	"fmt"
	"time"
)

var creators = make(map[Name]Creator)

// Name is a name of transport.
type Name string

const (
	Polling   Name = "polling"
	Websocket Name = "websocket"
	SSE       Name = "sse"
)

// Register registers a transport's creator. The creator is used when creates a transport with it's name.
func Register(name Name, creator Creator) {
	creators[name] = creator
}

// Create creates a transport with the name.
func Create(name Name, pingTimeout time.Duration, callbacks Callbacks) (Transport, error) {
	creator, ok := creators[name]
	if !ok {
		return nil, fmt.Errorf("no transport with name %s", name)
	}

	return creator(pingTimeout, callbacks), nil
}
