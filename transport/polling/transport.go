package polling

import (
	"io"
	"net/http"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/transport"
)

type pTransport struct {
	connChan chan base.Conn
}

// New creates a new polling transport.
func New() transport.Transport {
	return &pTransport{
		connChan: make(chan base.Conn),
	}
}

func (s *pTransport) ConnChan() <-chan base.Conn {
	return s.connChan
}

func (s *pTransport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	closed := make(chan struct{})
	conn := newServerConn(r, closed)
	s.connChan <- conn
	handler := conn.(http.Handler)
	handler.ServeHTTP(w, r)
}

func retError(url, op string, err error) error {
	if err == nil || err == io.EOF {
		return err
	}
	if opErr, ok := err.(*base.OpError); ok {
		return opErr
	}
	return base.OpErr(url, op, err)
}
