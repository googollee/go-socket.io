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

type server struct {
	upgrader websocket.Upgrader
	connChan chan base.Conn
}

// NewServer creates new websocket transport server.
func NewServer(c *Configure) transport.Transport {
	if c == nil {
		c = DefaultConfigure
	}
	return &server{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  c.ReadBufferSize,
			WriteBufferSize: c.WriteBufferSize,
		},
		connChan: make(chan base.Conn),
	}
}

func (s *server) ConnChan() <-chan base.Conn {
	return s.connChan
}

func (s *server) ServeHTTP(header http.Header, w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer c.Close()

	closed := make(chan struct{})
	s.connChan <- newConn(c, closed)
	<-closed
}
