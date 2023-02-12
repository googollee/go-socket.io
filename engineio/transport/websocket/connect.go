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

type Connection struct {
	transport.FrameReader
	transport.FrameWriter

	ws wrapper

	url          url.URL
	remoteHeader http.Header

	closed    chan struct{}
	closeOnce sync.Once
}

func newConn(ws *websocket.Conn, url url.URL, header http.Header) *Connection {
	w := newWrapper(ws)

	return &Connection{
		url:          url,
		remoteHeader: header,
		ws:           w,
		closed:       make(chan struct{}),
		FrameReader:  packet.NewDecoder(w),
		FrameWriter:  packet.NewEncoder(w),
	}
}

func (c *Connection) URL() url.URL {
	return c.url
}

func (c *Connection) RemoteHeader() http.Header {
	return c.remoteHeader
}

func (c *Connection) LocalAddr() net.Addr {
	return c.ws.LocalAddr()
}

func (c *Connection) RemoteAddr() net.Addr {
	return c.ws.RemoteAddr()
}

func (c *Connection) SetReadDeadline(t time.Time) error {
	return c.ws.SetReadDeadline(t)
}

func (c *Connection) SetWriteDeadline(t time.Time) error {
	// TODO: is locking really needed for SetWriteDeadline? If so, what about
	// the read deadline?
	c.ws.writeLocker.Lock()
	err := c.ws.SetWriteDeadline(t)
	c.ws.writeLocker.Unlock()

	return err
}

func (c *Connection) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	<-c.closed
}

func (c *Connection) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return c.ws.Close()
}
