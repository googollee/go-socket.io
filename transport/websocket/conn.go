package websocket

import (
	"net"
	"sync"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/packet"
	"github.com/gorilla/websocket"
)

type conn struct {
	ws        wrapper
	closed    chan struct{}
	closeOnce sync.Once
	base.FrameWriter
	base.FrameReader
}

func newConn(ws *websocket.Conn, closed chan struct{}) base.Conn {
	w := newWrapper(ws)
	return &conn{
		ws:          w,
		closed:      closed,
		FrameReader: packet.NewDecoder(w),
		FrameWriter: packet.NewEncoder(w),
	}
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
