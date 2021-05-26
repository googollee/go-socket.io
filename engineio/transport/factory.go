package transport

import (
	"fmt"
	"time"
)

var creators = make(map[Name]Creator)

type Name string

const (
	Polling   Name = "polling"
	Websocket      = "websocket"
	SSE            = "sse"
)

func Register(name Name, creator Creator) {
	creators[name] = creator
}

func Create(name Name, pingTimeout time.Duration, callbacks Callbacks) (Transport, error) {
	creator, ok := creators[name]
	if !ok {
		return nil, fmt.Errorf("no transport with name %s", name)
	}

	return creator(pingTimeout, callbacks), nil
}
