package websocket

import (
	"net/http"

	"github.com/googollee/go-engine.io/base"
	"github.com/gorilla/websocket"
)

type DialError struct {
	error
	Response *http.Response
}

type Dialer websocket.Dialer

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

	return newConn(c, closed), nil
}
