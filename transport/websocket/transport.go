package websocket

import (
	"net/http"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/transport"
	"github.com/gorilla/websocket"
)

// Configure is configure of websocket transport.
type Configure struct {
	ReadBufferSize  int
	WriteBufferSize int
}

// DefaultConfigure is default.
var DefaultConfigure = &Configure{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type wsTransport struct {
	upgrader websocket.Upgrader
	connChan chan base.Conn
}

// New creates new websocket transport.
func New(c *Configure) transport.Transport {
	if c == nil {
		c = DefaultConfigure
	}
	return &wsTransport{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  c.ReadBufferSize,
			WriteBufferSize: c.WriteBufferSize,
		},
		connChan: make(chan base.Conn),
	}
}

func (s *wsTransport) ConnChan() <-chan base.Conn {
	return s.connChan
}

func (s *wsTransport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer c.Close()

	closed := make(chan struct{})
	s.connChan <- newConn(c, r.Header, closed)
	<-closed
}
