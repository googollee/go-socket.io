package websocket

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/googollee/go-socket.io/engineio/transport/utils"
	"github.com/gorilla/websocket"
)

type Connect struct {
	dialer *websocket.Dialer

	// options
	ReadBufferSize  int
	WriteBufferSize int

	Subprotocols     []string
	TLSClientConfig  *tls.Config
	HandshakeTimeout time.Duration

	Proxy       func(*http.Request) (*url.URL, error)
	NetDial     func(network, addr string) (net.Conn, error)
	CheckOrigin func(r *http.Request) bool
}

func New() *Connect {
	return &Connect{
		dialer: &websocket.Dialer{
			//ReadBufferSize:   t.ReadBufferSize,
			//WriteBufferSize:  t.WriteBufferSize,
			//NetDial:          t.NetDial,
			//Proxy:            t.Proxy,
			//TLSClientConfig:  t.TLSClientConfig,
			//HandshakeTimeout: t.HandshakeTimeout,
			//Subprotocols:     t.Subprotocols,
		},
	}
}

func (t *Connect) Do(req *http.Request) (*Connect, error) {
	switch req.URL.Scheme {
	case "http":
		req.URL.Scheme = "ws"
	case "https":
		req.URL.Scheme = "wss"
	}

	req.URL.Query().Set("transport", "websocket")
	req.URL.Query().Set("t", utils.Timestamp())

	conn, resp, err := t.dialer.Dial(req.URL.String(), req.Header)
	if err != nil {
		return nil, err
	}

	return newConn(conn, resp), nil
}
