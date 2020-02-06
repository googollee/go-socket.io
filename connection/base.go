package engineio

import (
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/googollee/go-socket.io/connection/base"
)

// FrameType is type of a message frame.
type FrameType base.FrameType

const (
	// TEXT is text type message.
	TEXT = FrameType(base.FrameString)
	// BINARY is binary type message.
	BINARY = FrameType(base.FrameBinary)
)

// Conn is connection.
type Conn interface {
	ID() string
	NextReader() (FrameType, io.ReadCloser, error)
	NextWriter(typ FrameType) (io.WriteCloser, error)
	Close() error
	URL() url.URL
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	RemoteHeader() http.Header
	SetContext(v interface{})
	Context() interface{}
}
