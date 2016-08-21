package websocket

import (
	"net"
	"net/http"
	"sync"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/packet"
	"github.com/gorilla/websocket"
)

type conn struct {
	remoteHeader http.Header
	ws           wrapper
	closed       chan struct{}
	closeOnce    sync.Once
	base.FrameWriter
	base.FrameReader
}

func newConn(ws *websocket.Conn, remote http.Header, closed chan struct{}) base.Conn {
	w := newWrapper(ws)
	return &conn{
		remoteHeader: remote,
		ws:           w,
		closed:       closed,
		FrameReader:  packet.NewDecoder(w),
		FrameWriter:  packet.NewEncoder(w),
	}
}

func (c *conn) RemoteHeader() http.Header {
	return c.remoteHeader
}

func (c *conn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "invalid websocket request", http.StatusInternalServerError)
}

func (c *conn) LocalAddr() net.Addr {
	return c.ws.LocalAddr()
}

func (c *conn) RemoteAddr() net.Addr {
	return c.ws.RemoteAddr()
}

func (c *conn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return c.ws.Close()
}
