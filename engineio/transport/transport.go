package transport

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/googollee/go-socket.io/engineio/packet"
)

// FrameReader reads a frame. It need be closed before next reading.
type FrameReader interface {
	NextReader() (packet.FrameType, packet.PacketType, io.ReadCloser, error)
}

// FrameWriter writes a frame. It need be closed before next writing.
type FrameWriter interface {
	NextWriter(ft packet.FrameType, pt packet.PacketType) (io.WriteCloser, error)
}

// Conn is a transport connection.
type Conn interface {
	FrameReader
	FrameWriter
	io.Closer
	URL() url.URL
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	RemoteHeader() http.Header
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}
