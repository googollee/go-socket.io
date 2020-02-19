package polling

import (
	"bytes"
	"html/template"
	"net"
	"net/http"
	"net/url"

	"github.com/googollee/go-socket.io/connection/base"
	"github.com/googollee/go-socket.io/connection/payload"
)

type serverConn struct {
	*payload.Payload
	supportBinary bool

	remoteHeader http.Header
	localAddr    Addr
	remoteAddr   Addr
	url          url.URL
	jsonp        string
}

func newServerConn(r *http.Request) base.Conn {
	query := r.URL.Query()
	supportBinary := query.Get("b64") == ""
	jsonp := query.Get("j")
	if jsonp != "" {
		supportBinary = false
	}
	return &serverConn{
		Payload:       payload.New(supportBinary),
		supportBinary: supportBinary,
		remoteHeader:  r.Header,
		localAddr:     Addr{r.Host},
		remoteAddr:    Addr{r.RemoteAddr},
		url:           *r.URL,
		jsonp:         jsonp,
	}
}

func (c *serverConn) URL() url.URL {
	return c.url
}

func (c *serverConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *serverConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *serverConn) RemoteHeader() http.Header {
	return c.remoteHeader
}

func (c *serverConn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if jsonp := r.URL.Query().Get("j"); jsonp != "" {
			buf := bytes.NewBuffer(nil)
			if err := c.Payload.FlushOut(buf); err != nil {
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
		if err := c.Payload.FlushOut(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	case "POST":
		mime := r.Header.Get("Content-Type")
		supportBinary, err := mimeSupportBinary(mime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := c.Payload.FeedIn(r.Body, supportBinary); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Write([]byte("ok"))
		return
	default:
		http.Error(w, "invalid method", http.StatusBadRequest)
	}
}
