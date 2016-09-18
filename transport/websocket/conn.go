package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/packet"
	"github.com/gorilla/websocket"
)

type conn struct {
	url          string
	remoteHeader http.Header
	ws           wrapper
	closed       chan struct{}
	closeOnce    sync.Once
	base.FrameWriter
	base.FrameReader
}

func newConn(ws *websocket.Conn, url string, header http.Header) base.Conn {
	w := newWrapper(ws)
	closed := make(chan struct{})
	return &conn{
		url:          url,
		remoteHeader: header,
		ws:           w,
		closed:       closed,
		FrameReader:  packet.NewDecoder(w),
		FrameWriter:  packet.NewEncoder(w),
	}
}

func (c *conn) URL() string {
	return c.url
}

func (c *conn) RemoteHeader() http.Header {
	return c.remoteHeader
}

func (c *conn) LocalAddr() string {
	return c.ws.LocalAddr().String()
}

func (c *conn) RemoteAddr() string {
	return c.ws.RemoteAddr().String()
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return c.ws.SetReadDeadline(t)
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return c.ws.SetWriteDeadline(t)
}

func (c *conn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	<-c.closed
}

func (c *conn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return c.ws.Close()
}
