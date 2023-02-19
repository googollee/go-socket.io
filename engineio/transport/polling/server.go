package polling

import (
	"bytes"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/googollee/go-socket.io/engineio/payload"
)

type serverConn struct {
	*payload.Payload
	transport     *Transport
	supportBinary bool

	remoteHeader http.Header
	localAddr    Addr
	remoteAddr   Addr
	url          url.URL
	jsonp        string
}

func newServerConn(t *Transport, r *http.Request) *serverConn {
	query := r.URL.Query()
	jsonp := query.Get("j")
	supportBinary := query.Get("b64") == ""
	if jsonp != "" {
		supportBinary = false
	}

	return &serverConn{
		Payload:       payload.New(supportBinary),
		transport:     t,
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

func (c *serverConn) SetHeaders(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.UserAgent(), ";MSIE") || strings.Contains(r.UserAgent(), "Trident/") {
		w.Header().Set("X-XSS-Protection", "0")
	}

	// just in case the default behaviour gets changed and it has to handle an origin check
	checkOrigin := Default.CheckOrigin
	if c.transport.CheckOrigin != nil {
		checkOrigin = c.transport.CheckOrigin
	}

	if checkOrigin != nil && checkOrigin(r) {
		if r.URL.Query().Get("j") == "" {
			origin := r.Header.Get("Origin")
			if origin == "" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
	}
}

func (c *serverConn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodOptions:
		if r.URL.Query().Get("j") == "" {
			c.SetHeaders(w, r)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(200)
		}

	case http.MethodGet:
		c.SetHeaders(w, r)

		if jsonp := r.URL.Query().Get("j"); jsonp != "" {
			buf := bytes.NewBuffer(nil)
			if err := c.Payload.FlushOut(buf); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/javascript; charset=UTF-8")
			pl := template.JSEscapeString(buf.String())

			_, _ = w.Write([]byte("___eio[" + jsonp + "](\""))
			_, _ = w.Write([]byte(pl))
			_, _ = w.Write([]byte("\");"))

			return
		}
		if c.supportBinary {
			w.Header().Set("Content-Type", "application/octet-stream")
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		}

		if err := c.Payload.FlushOut(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:
		c.SetHeaders(w, r)

		mime := r.Header.Get("Content-Type")
		isSupportBinary, err := mimeIsSupportBinary(mime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := c.Payload.FeedIn(r.Body, isSupportBinary); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = w.Write([]byte("ok"))
		if err != nil {
			fmt.Printf("ack post err=%s\n", err.Error())
		}

	default:
		http.Error(w, "invalid method", http.StatusBadRequest)
	}
}
