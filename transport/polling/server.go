package polling

import (
	"bytes"
	"html/template"
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

	query         url.Values
	closed        chan struct{}
	closeOnce     sync.Once
	remoteHeader  http.Header
	localAddr     string
	remoteAddr    string
	url           url.URL
	supportBinary bool
	jsonp         string
}

func newServerConn(r *http.Request, closed chan struct{}) base.Conn {
	query := r.URL.Query()
	supportBinary := query.Get("b64") == ""
	jsonp := query.Get("j")
	if jsonp != "" {
		supportBinary = false
	}
	ret := &serverConn{
		closed:        closed,
		remoteHeader:  r.Header,
		localAddr:     r.Host,
		remoteAddr:    r.RemoteAddr,
		url:           *r.URL,
		supportBinary: supportBinary,
		jsonp:         jsonp,
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
		if jsonp := r.URL.Query().Get("j"); jsonp != "" {
			buf := bytes.NewBuffer(nil)
			if err := c.encoder.FlushOut(buf); err != nil {
				c.storeErr("flush out", err)
				c.Close()
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/javascript; charset=UTF-8")
			pl := template.JSEscapeString(buf.String())
			w.Write([]byte("___eio[" + jsonp + "](\""))
			w.Write([]byte(pl))
			w.Write([]byte("\");"))
			return
		}
		if c.supportBinary {
			w.Header().Set("Content-Type", "application/octet-stream")
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		}
		if err := c.encoder.FlushOut(w); err != nil {
			c.storeErr("flush out", err)
			c.Close()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	case "POST":
		mime := r.Header.Get("Content-Type")
		typ, err := normalizeMime(mime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := c.decoder.FeedIn(typ, r.Body); err != nil {
			c.storeErr("feed in", err)
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

func (c *serverConn) storeErr(op string, err error) error {
	if err == nil {
		return err
	}
	if _, ok := err.(*base.OpError); ok || err == io.EOF {
		return err
	}
	return c.err.Store(base.OpErr(c.url.String(), op, err))
}
