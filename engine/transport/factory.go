package transport

import (
	"fmt"
	"time"
)

var creators = make(map[string]Creator)

func Register(name string, creator Creator) {
	creators[name] = creator
}

func Create(name string, pingTimeout time.Duration, callbacks Callbacks) (Transport, error) {
	creator, ok := creators[name]
	if !ok {
		return nil, fmt.Errorf("no transport with name %s", name)
	}

	return creator(pingTimeout, callbacks), nil
}
