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

// Creator is a function to create a transport.
// In v4, pingTimeout is a duration of pingInterval.
// In v3, pingTimeout is a duration of pingTimeout.
type Creator func(pingTimeout time.Duration, alloc BufferAllocator, callbacks Callbacks) Transport

// Register registers a transport's creator. The creator is used when creates a transport with it's name.
func Register(name Name, creator Creator) {
	creators[name] = creator
}

// BufferAllocator returns a buffer.
// It uses to allocate a buffer for a new transport. The size is decided by the provider.
// It could use a sync.Pool to reuse the buffer.
type BufferAllocator interface {
	// New returns a buffer.
	New() []byte
	// Free frees a buffer.
	Free([]byte)
}

// Create creates a transport with the name.
func Create(name Name, pingTimeout time.Duration, allocator BufferAllocator, callbacks Callbacks) (Transport, error) {
	creator, ok := creators[name]
	if !ok {
		return nil, fmt.Errorf("no transport with name %s", name)
	}

	return creator(pingTimeout, allocator, callbacks), nil
}
