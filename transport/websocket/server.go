package websocket

import (
	"net/http"

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
	connChan chan transport.ConnArg
}

// NewServer creates new websocket transport server.
func NewServer(c *Configure) transport.Transport {
	return &server{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  c.ReadBufferSize,
			WriteBufferSize: c.WriteBufferSize,
		},
		connChan: make(chan transport.ConnArg),
	}
}

func (s *server) ConnChan() <-chan transport.ConnArg {
	return s.connChan
}

func (s *server) ServeHTTP(sid string, header http.Header, w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer c.Close()

	closed := make(chan struct{})
	arg := transport.ConnArg{
		Conn:  newConn(sid, c, closed),
		Close: closed,
	}
	s.connChan <- arg
	<-closed
}
