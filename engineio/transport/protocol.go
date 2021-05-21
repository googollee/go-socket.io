package transport

import (
	"io"
	"net/http"
	"time"
)

// Transport is a transport to maintain a connection. It assumes that all
// requests are related with this transport. The holder of transports needs
// to send requests to the correct transport.
// All methods of a transport could be called from different goroutines.
type Transport interface {
	// Name returns the name of this transport, e.g. polling/websocket.
	Name() string

	// Close closes this transport.
	Close() error

	// SendFrame creates an frame writer to send an frame.
	SendFrame() (FrameWriter, error)

	// ServeHTTP serves HTTP requests.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// FrameReader provides a reader to read a frame.
type FrameReader interface {
	io.Reader

	// ReadByte reads a byte.
	ReadByte() (byte, error)
}

// FrameWriter provides a writer to write a frame.
type FrameWriter interface {
	io.Writer

	// WriteByte writes a byte.
	WriteByte(b byte) error
}

// Callbacks is a group of callback functions which a transport may call
// during processing data.
type Callbacks interface {
	// OnPingTimeout is called when pingTimeout reachs.
	// In v4, it should send a ping message with the transport t.
	// In v3, it should close the transport t.
	OnPingTimeout(t Transport)

	// OnFrame is called when the transport t receives a frame.
	// The HTTP request of that frame is req.
	OnFrame(t Transport, req *http.Request, rd FrameReader) error

	// OnError is called when the transport meets an error.
	// It could be an error when processing receiving data, or an error when
	// sending data.
	// The call of OnError could be in different goroutines.
	OnError(t Transport, err error)
}

// Creator is a function to create a transport.
// In v4, pingTimeout is a duration of pingInterval.
// In v3, pingTimeout is a duration of pingTimeout.
type Creator func(pingTimeout time.Duration, callbacks Callbacks) Transport
