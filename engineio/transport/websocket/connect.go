package websocket

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/googollee/go-socket.io/engineio/packet"
)

type options struct {
	readBufferSize  int
	writeBufferSize int

	subProtocols      []string
	tlsClientConfig   *tls.Config
	handshakeTimeout  time.Duration
	enableCompression bool

	proxy       func(*http.Request) (*url.URL, error)
	netDial     func(network, addr string) (net.Conn, error)
	checkOrigin func(r *http.Request) bool
}

type OptionFunc func(o *options)

func New(w http.ResponseWriter, r *http.Request, opts ...OptionFunc) (*Connection, error) {
	var o options

	for _, opt := range opts {
		opt(&o)
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout: o.handshakeTimeout,
		ReadBufferSize:   o.readBufferSize,
		WriteBufferSize:  o.writeBufferSize,
		//WriteBufferPool:   o.writeBufferSize,
		Subprotocols: o.subProtocols,
		//Error:             o.err,
		CheckOrigin:       o.checkOrigin,
		EnableCompression: o.enableCompression,
	}

	conn, err := upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		return nil, err
	}

	return newConn(conn, *r.URL, r.Header), nil
}

type Connection struct {
	*packet.Decoder
	*packet.Encoder
	//transport.FrameReader
	//transport.FrameWriter

	ws wrapper

	url          url.URL
	remoteHeader http.Header

	closed    chan struct{}
	closeOnce sync.Once
}

//wrapper
//NextReader() (frame.Type, io.ReadCloser, error)
//NextWriter(FType frame.Type) (io.WriteCloser, error)

func newConn(ws *websocket.Conn, url url.URL, header http.Header) *Connection {
	w := newWrapper(ws)

	return &Connection{
		url:          url,
		remoteHeader: header,
		ws:           w,
		closed:       make(chan struct{}),
		Decoder:      packet.NewDecoder(w),
		Encoder:      packet.NewEncoder(w),
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
