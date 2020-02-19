package engineio

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/googollee/go-socket.io/connection/base"
	"github.com/googollee/go-socket.io/connection/transport"
)

// Dialer is dialer configure.
type Dialer struct {
	Transports []transport.Transport
}

// Dial returns a connection which dials to url with requestHeader.
func (d *Dialer) Dial(urlStr string, requestHeader http.Header) (Conn, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	query.Set("EIO", "3")
	u.RawQuery = query.Encode()
	var conn base.Conn
	for i := len(d.Transports) - 1; i >= 0; i-- {
		if conn != nil {
			conn.Close()
		}
		t := d.Transports[i]
		conn, err = t.Dial(u, requestHeader)
		if err != nil {
			continue
		}
		var params base.ConnParameters
		if p, ok := conn.(transport.Opener); ok {
			params, err = p.Open()
			if err != nil {
				continue
			}
		} else {
			var pt base.PacketType
			var r io.ReadCloser
			_, pt, r, err = conn.NextReader()
			if err != nil {
				continue
			}
			func() {
				defer r.Close()
				if pt != base.OPEN {
					err = errors.New("invalid open")
					return
				}
				params, err = base.ReadConnParameters(r)
				if err != nil {
					return
				}
			}()
		}
		if err != nil {
			continue
		}
		ret := &client{
			conn:      conn,
			params:    params,
			transport: t.Name(),
			close:     make(chan struct{}),
		}
		go ret.serve()
		return ret, nil
	}
	return nil, err
}

type client struct {
	conn      base.Conn
	params    base.ConnParameters
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

func (c *client) NextReader() (FrameType, io.ReadCloser, error) {
	for {
		ft, pt, r, err := c.conn.NextReader()
		if err != nil {
			return 0, nil, err
		}
		switch pt {
		case base.PONG:
			c.conn.SetReadDeadline(time.Now().Add(c.params.PingTimeout))
		case base.CLOSE:
			c.Close()
			return 0, nil, io.EOF
		case base.MESSAGE:
			return FrameType(ft), r, nil
		}
		r.Close()
	}
}

func (c *client) NextWriter(typ FrameType) (io.WriteCloser, error) {
	return c.conn.NextWriter(base.FrameType(typ), base.MESSAGE)
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
	defer c.conn.Close()
	for {
		select {
		case <-c.close:
			return
		case <-time.After(c.params.PingInterval):
		}
		w, err := c.conn.NextWriter(base.FrameString, base.PING)
		if err != nil {
			return
		}
		if err := w.Close(); err != nil {
			return
		}
		c.conn.SetWriteDeadline(time.Now().Add(c.params.PingTimeout))
	}
}
