package polling

import (
	"bytes"
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

type clientConn struct {
	supportBinary bool
	retry         int
	request       http.Request
	remoteHeader  atomic.Value
	httpClient    *http.Client
	closeOnce     sync.Once
	encoder       payload.Encoder
	decoder       payload.Decoder
	signal        *payload.Signal
}

func (c *clientConn) SetReadDeadline(t time.Time) error {
	err := c.decoder.SetDeadline(t)
	if err == nil {
		return nil
	}
	return base.OpErr(c.request.URL.String(), "set read deadline", err)
}

func (c *clientConn) NextReader() (base.FrameType, base.PacketType, io.Reader, error) {
	ft, pt, r, err := c.decoder.NextReader()
	if err != nil {
		c.Close()
	}
	return ft, pt, r, retError(c.request.URL.String(), "read", err)
}

func (c *clientConn) SetWriteDeadline(t time.Time) error {
	err := c.encoder.SetDeadline(t)
	if err == nil {
		return nil
	}
	return base.OpErr(c.request.URL.String(), "set write deadline", err)
}

func (c *clientConn) NextWriter(ft base.FrameType, pt base.PacketType) (io.WriteCloser, error) {
	w, err := c.encoder.NextWriter(ft, pt)
	if err != nil {
		c.Close()
	}
	return w, retError(c.request.URL.String(), "write", err)
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
		c.signal.Close()
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
		if err := c.encoder.FlushOut(buf); err != nil {
			c.storeErr("flush out", err)
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
			c.storeErr("post(write) to", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			c.storeErr("post(write) to ",
				fmt.Errorf("invalid response(%d): %s", resp.StatusCode, resp.Status))
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
			c.storeErr("get(read) from", err)
			return
		}

		func() {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				c.storeErr("get(read) from",
					fmt.Errorf("invalid response(%d): %s", resp.StatusCode, resp.Status))
				run = false
				return
			}
			if c.remoteHeader.Load() == nil {
				c.remoteHeader.Store(resp.Header)
			}
			mime := resp.Header.Get("Content-Type")
			typ, err := normalizeMime(mime)
			if err != nil {
				c.storeErr("get(read) from", err)
				run = false
				return
			}
			if err := c.decoder.FeedIn(typ, resp.Body); err != nil {
				c.storeErr("feed in", err)
				run = false
				return
			}
		}()

		if init {
			break
		}
	}
}

func (c *clientConn) storeErr(op string, err error) error {
	if err == nil {
		return err
	}
	if _, ok := err.(*base.OpError); ok || err == io.EOF {
		return err
	}
	return c.signal.StoreError(base.OpErr(c.request.URL.String(), op, err))
}
