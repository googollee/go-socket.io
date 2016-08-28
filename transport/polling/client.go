package polling

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/payload"
)

var DefaultDialer = &Dialer{
	Retry: 3,
}

type Dialer struct {
	Client *http.Client
	Retry  int
}

func (d *Dialer) Dial(url string, requestHeader http.Header) (base.Conn, error) {
	ret, err := d.dial(url, requestHeader)
	if err != nil {
		return nil, err
	}

	go ret.doGet(false)
	go ret.doPost()

	return ret, nil
}

func (d *Dialer) Open(url string, requestHeader http.Header) (base.ConnParameters, error) {
	c, err := d.dial(url, requestHeader)
	if err != nil {
		return base.ConnParameters{}, base.OpErr(url, "dial", err)
	}
	defer c.Close()

	go c.doGet(true)

	_, pt, r, err := c.NextReader()
	if err != nil {
		return base.ConnParameters{}, base.OpErr(url, "open", err)
	}
	if pt != base.OPEN {
		return base.ConnParameters{}, base.OpErr(url, "open", errors.New("not open packet"))
	}
	return base.ReadConnParameters(r)
}

func (d *Dialer) dial(url string, requestHeader http.Header) (*clientConn, error) {
	if d.Client == nil {
		d.Client = &http.Client{}
	}
	if d.Retry == 0 {
		d.Retry = 3
	}
	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, base.OpErr(url, "create request", err)
	}
	for k, v := range requestHeader {
		req.Header[k] = v
	}
	supportBinary := req.URL.Query().Get("b64") == ""
	closed := make(chan struct{})
	if supportBinary {
		req.Header.Set("Content-Type", "application/octet-stream")
	} else {
		req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	}

	ret := &clientConn{
		supportBinary: supportBinary,
		retry:         d.Retry,
		request:       *req,
		httpClient:    d.Client,
		closed:        closed,
	}
	ret.err.Store(base.OpErr(url, "i/o", io.EOF))
	ret.Encoder = payload.NewEncoder(supportBinary, closed, &ret.err)
	ret.Decoder = payload.NewDecoder(closed, &ret.err)

	return ret, nil
}

type clientConn struct {
	supportBinary bool
	retry         int
	request       http.Request
	remoteHeader  atomic.Value
	httpClient    *http.Client
	closed        chan struct{}
	closeOnce     sync.Once
	err           atomic.Value
	payload.Encoder
	payload.Decoder
}

func (c *clientConn) SetReadDeadline(t time.Time) error {
	err := c.Decoder.SetDeadline(t)
	if err == nil {
		return nil
	}
	return base.OpErr(c.request.URL.String(), "set read deadline", err)
}

func (c *clientConn) SetWriteDeadline(t time.Time) error {
	err := c.Encoder.SetDeadline(t)
	if err != nil {
		return nil
	}
	return base.OpErr(c.request.URL.String(), "set write deadline", err)
}

func (c *clientConn) LocalAddr() string {
	return ""
}

func (c *clientConn) RemoteAddr() string {
	return c.request.Host
}

func (c *clientConn) RemoteHeader() http.Header {
	v := c.remoteHeader.Load()
	if v == nil {
		return nil
	}
	return v.(http.Header)
}

func (c *clientConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *clientConn) doPost() {
	defer c.Close()

	buf := bytes.NewBuffer(nil)
	rc := ioutil.NopCloser(buf)
	req := c.request
	req.Method = "POST"
	req.Body = rc
	for {
		buf.Reset()
		if err := c.Encoder.FlushOut(buf); err != nil {
			c.err.Store(base.OpErr(c.request.URL.String(), "flush out", err))
			return
		}

		var resp *http.Response
		var err error
		for i := 0; i < c.retry; i++ {
			resp, err = c.httpClient.Do(&req)
			if err == nil {
				break
			}
		}
		if err != nil {
			c.err.Store(base.OpErr(c.request.URL.String(), "post(write) to", err))
			return
		}

		if resp.StatusCode != http.StatusOK {
			c.err.Store(base.OpErr(
				c.request.URL.String(),
				"post(write) to ",
				fmt.Errorf("invalid response(%d): %s", resp.StatusCode, resp.Status)))
			return
		}
	}
}

func (c *clientConn) doGet(init bool) {
	defer c.Close()

	req := c.request
	req.Method = "GET"
	for run := true; run; {
		var resp *http.Response
		var err error
		for i := 0; i < c.retry; i++ {
			resp, err = c.httpClient.Do(&req)
			if err == nil {
				break
			}
		}
		if err != nil {
			c.err.Store(base.OpErr(c.request.URL.String(), "get(read) from", err))
			return
		}

		func() {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				c.err.Store(base.OpErr(
					c.request.URL.String(),
					"get(read) from",
					fmt.Errorf("invalid response(%d): %s", resp.StatusCode, resp.Status)))
				run = false
				return
			}
			if c.remoteHeader.Load() == nil {
				c.remoteHeader.Store(resp.Header)
			}
			mime := resp.Header.Get("Content-Type")
			var typ base.FrameType
			switch mime {
			case "text/plain;charset=UTF-8":
				typ = base.FrameString
			case "application/octet-stream":
				typ = base.FrameBinary
			default:
				c.err.Store(base.OpErr(
					c.request.URL.String(),
					"get(read) from",
					errors.New("invalid content-type")))
				run = false
				return
			}
			if err := c.Decoder.FeedIn(typ, resp.Body); err != nil {
				c.err.Store(base.OpErr(c.request.URL.String(), "feed in", err))
				run = false
				return
			}
		}()

		if init {
			break
		}
	}
}
