package polling

import (
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/payload"
)

type serverConn struct {
	encoder payload.Encoder
	decoder payload.Decoder
	err     payload.AtomicError

	readDeadline  time.Time
	writeDeadline time.Time

	query        url.Values
	closed       chan struct{}
	closeOnce    sync.Once
	remoteHeader http.Header
	localAddr    string
	remoteAddr   string
	url          url.URL
}

func newServerConn(r *http.Request, closed chan struct{}) base.Conn {
	supportBinary := r.Header.Get("b64") == ""
	ret := &serverConn{
		closed:       closed,
		remoteHeader: r.Header,
		localAddr:    r.Host,
		remoteAddr:   r.RemoteAddr,
		url:          *r.URL,
	}
	ret.encoder = payload.NewEncoder(supportBinary, closed, &ret.err)
	ret.decoder = payload.NewDecoder(closed, &ret.err)
	return ret
}

func (c *serverConn) SetReadDeadline(t time.Time) error {
	err := c.decoder.SetDeadline(t)
	if err == nil {
		return nil
	}
	return base.OpErr(c.url.String(), "SetReadDeadline", err)
}

func (c *serverConn) NextReader() (base.FrameType, base.PacketType, io.Reader, error) {
	ft, pt, r, err := c.decoder.NextReader()
	return ft, pt, r, retError(c.url.String(), "read", err)
}

func (c *serverConn) SetWriteDeadline(t time.Time) error {
	err := c.encoder.SetDeadline(t)
	if err != nil {
		return nil
	}
	return base.OpErr(c.url.String(), "SetWriteDeadline", err)
}

func (c *serverConn) NextWriter(ft base.FrameType, pt base.PacketType) (io.WriteCloser, error) {
	w, err := c.encoder.NextWriter(ft, pt)
	return w, retError(c.url.String(), "write", err)
}

func (c *serverConn) LocalAddr() string {
	return c.localAddr
}

func (c *serverConn) RemoteAddr() string {
	return c.remoteAddr
}

func (c *serverConn) RemoteHeader() http.Header {
	return c.remoteHeader
}

func (c *serverConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *serverConn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if err := c.encoder.FlushOut(w); err != nil {
			c.err.Store(base.OpErr(c.url.String(), "flush out", err))
			c.Close()
		}
		return
	case "POST":
		mime := r.Header.Get("Content-Type")
		var typ base.FrameType
		switch mime {
		case "text/plain;charset=UTF-8":
			typ = base.FrameString
		case "application/octet-stream":
			typ = base.FrameBinary
		default:
			http.Error(w, "invalid content-type", http.StatusBadRequest)
			return
		}
		if err := c.decoder.FeedIn(typ, r.Body); err != nil {
			c.err.Store(base.OpErr(c.url.String(), "feed in", err))
			c.Close()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Write([]byte("ok"))
		return
	default:
		http.Error(w, "invalid method", http.StatusBadRequest)
	}
}
