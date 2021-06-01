package engineio

import (
	"io"

	"github.com/googollee/go-socket.io/engineio/frame"
)

// Session represents a engineio session. A session is a network connection with a id.
// Session methods could be called in any goroutine.
type Session interface {
	ID() string
	Transport() string
	RemoteIP() string
	LocalIP() string

	Close() error
	Store(key string, value interface{})
	Get(key string) interface{}

	// SendFrame should be called after closing last frame.
	SendFrame(frame.Type) (io.WriteCloser, error)
}
