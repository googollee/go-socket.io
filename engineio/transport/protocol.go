package transport

import (
	"io"
	"net/http"

	"github.com/googollee/go-socket.io/engineio/frame"
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
	SendFrame(frame.Type) (io.WriteCloser, error)

	// PrepareHTTP prepares the transport with the first http requests.
	// If PrepareHTTP() returns an error, the transport will be closed and can't serve other HTTP requests.
	PrepareHTTP(w http.ResponseWriter, r *http.Request) error

	// ServeHTTP serves HTTP requests.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
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
	// If it returns an error, the transport may reject the request.
	OnFrame(t Transport, req *http.Request, ft frame.Type, rd io.Reader) error

	// OnError is called when the transport meets an error.
	// It could be an error when processing receiving data, or an error when
	// sending data.
	// The call of OnError could be in different goroutines.
	OnError(t Transport, err error)
}
