package client

import (
	"errors"
	"fmt"
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
)

// Pauser is connection which can be paused and resumes.
type Pauser interface {
	Pause()
	Resume()
}

// Opener is client connection which need receive open message first.
type Opener interface {
	Open() (transport.ConnParameters, error)
}

type Client struct {
	Transports []transport.Type
}

func NewClient(transports []transport.Type) *Client {
	return &Client{
		Transports: transports,
	}
}

func (c *Client) Do(req *http.Request) (transport.Conn, error) {
	if req == nil {
		return nil, errors.New("")
	}

	query := req.URL.Query()

	query.Set("EIO", "3")
	req.URL.RawQuery = query.Encode()

	var conn transport.Conn
	var err error

	for i := len(c.Transports) - 1; i >= 0; i-- {
		if conn != nil {
			conn.Close()
		}

		t := c.Transports[i]

		conn, err = t.Dial(req.URL, req.Header)
		if err != nil {
			continue
		}

		var params transport.ConnParameters
		if p, ok := conn.(Opener); ok {
			params, err = p.Open()
			if err != nil {
				continue
			}
		} else {
			var pt packet.Type
			var r io.ReadCloser

			_, pt, r, err = conn.NextReader()
			if err != nil {
				continue
			}

			func() {
				defer r.Close()
				if pt != packet.OPEN {
					err = errors.New("invalid open")
					return
				}
				params, err = transport.ReadConnParameters(r)
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
			c.Close()
			return 0, nil, io.EOF

		case packet.MESSAGE:
			return session.FrameType(ft), r, nil
		}

		r.Close()
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
	defer c.conn.Close()

	for {
		select {
		case <-c.close:
			return
		case <-time.After(c.params.PingInterval):
		}

		w, err := c.conn.NextWriter(frame.String, packet.PING)
		if err != nil {
			return
		}

		if err := w.Close(); err != nil {
			return
		}

		if err = c.conn.SetWriteDeadline(time.Now().Add(c.params.PingInterval + c.params.PingTimeout)); err != nil {
			fmt.Printf("set writer's deadline error,msg:%s\n", err.Error())
		}
	}
}
