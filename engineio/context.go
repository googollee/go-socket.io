package engineio

import (
	"context"
	"io"
	"net/http"

	"github.com/googollee/go-socket.io/engineio/packet"
)

// Context provides info for a HTTP request.
type Context struct {
	// Standard context
	context.Context

	// The instance of current session.
	Session Session

	// The http.Request of current HTTP request.
	// It may be different when using polling transport.
	Request *http.Request

	// The current packet infomation.
	Packet Packet
}

type Packet struct {
	Type packet.Type
	Body io.Reader
}

// Next calls following middlewares in engine.io framework.
func (c *Context) Next() {}
