package engineio

import (
	"io"
	"net/http"
)

// Conn is the connection object of engine.io.
type ConnNew interface {

	// Id returns the session id of connection.
	Id() string

	// Request returns the first http request when established connection.
	Request() *http.Request

	// Close closes the connection.
	Close() error

	// NextReader returns the next message type, reader. If no message received, it will block.
	NextReader() (MessageType, io.ReadCloser, error)

	// NextWriter returns the next message writer with given message type.
	NextWriter(messageType MessageType) (io.WriteCloser, error)
}

type connCallback interface {
	OnPacket(r *packetDecoder)
	OnClose()
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type serverCallback interface {
	Config() config
	Transports() transportsType
	OnClose(sid string)
}
