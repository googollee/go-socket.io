package websocket

import (
	"net/http"

	"github.com/googollee/go-engine.io/base"
	"github.com/gorilla/websocket"
)

// DialError is the error when dialing to a server. It saves Response from
// server.
type DialError struct {
	error
	Response *http.Response
}

// Dialer contains options for connecting to server.
type Dialer websocket.Dialer

// Dial creates a new client connection.
func (d *Dialer) Dial(url string, requestHeader http.Header) (base.Conn, error) {
	dialer := websocket.Dialer(*d)
	c, resp, err := dialer.Dial(url, requestHeader)
	if err != nil {
		return nil, DialError{
			error:    err,
			Response: resp,
		}
	}

	closed := make(chan struct{})

	return newConn(c, resp.Header, closed), nil
}
