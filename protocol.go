package engineio

import (
	"io"
	"net/http"

	"github.com/googollee/go-engine.io/base"
)

type FrameType base.FrameType

const (
	TEXT   = FrameType(base.FrameString)
	BINARY = FrameType(base.FrameBinary)
)

type Conn interface {
	ID() string
	NextReader() (FrameType, io.ReadCloser, error)
	NextWriter(typ FrameType) (io.WriteCloser, error)
	Close() error
	LocalAddr() string
	RemoteAddr() string
	RemoteHeader() http.Header
}
