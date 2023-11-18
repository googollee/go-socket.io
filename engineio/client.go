package engineio

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/packet"
	"github.com/googollee/go-socket.io/engineio/session"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/logger"
)

// Opener is client connection which need receive open message first.
type Opener interface {
	Open() (transport.ConnParameters, error)
}

type client struct {
	conn      transport.Conn
	params    transport.ConnParameters
	transport string
	context   interface{}
	close     chan struct{}
	closeOnce sync.Once
}

func (c *client) SetContext(v interface{}) {
	c.context = v
}

func (c *client) Context() interface{} {
	return c.context
}

func (c *client) ID() string {
	return c.params.SID
}

func (c *client) Transport() string {
	return c.transport
}

func (c *client) Close() error {
	c.closeOnce.Do(func() {
		close(c.close)
	})
	return c.conn.Close()
}

func (c *client) NextReader() (session.FrameType, io.ReadCloser, error) {
	for {
		ft, pt, r, err := c.conn.NextReader()
		if err != nil {
			return 0, nil, err
		}

		switch pt {
		case packet.PONG:
			if err = c.conn.SetReadDeadline(time.Now().Add(c.params.PingInterval + c.params.PingTimeout)); err != nil {
				return 0, nil, err
			}

		case packet.CLOSE:
			if err = c.Close(); err != nil {
				logger.Error("close client with packet close:", err)
			}

			return 0, nil, io.EOF

		case packet.MESSAGE:
			return session.FrameType(ft), r, nil
		}

		if err = r.Close(); err != nil {
			logger.Error("close reader:", err)
		}
	}
}

func (c *client) NextWriter(typ session.FrameType) (io.WriteCloser, error) {
	return c.conn.NextWriter(frame.Type(typ), packet.MESSAGE)
}

func (c *client) URL() url.URL {
	return c.conn.URL()
}

func (c *client) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *client) RemoteHeader() http.Header {
	return c.conn.RemoteHeader()
}

func (c *client) serve() {
	defer func() {
		if closeErr := c.conn.Close(); closeErr != nil {
			logger.Error("close connect:", closeErr)
		}
	}()

	for {
		select {
		case <-c.close:
			return
		case <-time.After(c.params.PingInterval):
		}

		w, err := c.conn.NextWriter(frame.String, packet.PING)
		if err != nil {
			logger.Error("get next writer with string frame and packet ping:", err)

			return
		}

		if err = w.Close(); err != nil {
			logger.Error("close writer:", err)

			return
		}

		if err = c.conn.SetWriteDeadline(time.Now().Add(c.params.PingInterval + c.params.PingTimeout)); err != nil {
			logger.Error("set writer deadline:", err)
		}
	}
}
