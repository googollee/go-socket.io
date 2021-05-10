package websocket

import (
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/googollee/go-socket.io/engineio/packet"
	"github.com/googollee/go-socket.io/engineio/transport"
)

// conn implements base.Conn
type conn struct {
	transport.FrameReader
	transport.FrameWriter

	ws wrapper

	url          url.URL
	remoteHeader http.Header

	closed    chan struct{}
	closeOnce sync.Once
}

func newConn(ws *websocket.Conn, url url.URL, header http.Header) *conn {
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

func (c *conn) URL() url.URL {
	return c.url
}

func (c *conn) RemoteHeader() http.Header {
	return c.remoteHeader
}

func (c *conn) LocalAddr() net.Addr {
	return c.ws.LocalAddr()
}

func (c *conn) RemoteAddr() net.Addr {
	return c.ws.RemoteAddr()
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return c.ws.SetReadDeadline(t)
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	// TODO: is locking really needed for SetWriteDeadline? If so, what about
	// the read deadline?
	c.ws.writeLocker.Lock()
	err := c.ws.SetWriteDeadline(t)
	c.ws.writeLocker.Unlock()

	return err
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
