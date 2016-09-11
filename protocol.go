package engineio

import (
	"io"
	"net/http"

	"github.com/googollee/go-engine.io/base"
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
	LocalAddr() string
	RemoteAddr() string
	RemoteHeader() http.Header
}
