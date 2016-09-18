package polling

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/payload"
)

type clientConn struct {
	*payload.Payload

	httpClient   *http.Client
	retry        int
	request      http.Request
	remoteHeader http.Header
}

func open(retry int, client *http.Client, url string, requestHeader http.Header) (base.Conn, base.ConnParameters, error) {
	if client == nil {
		client = &http.Client{}
	}
	if retry == 0 {
		retry = 3
	}
	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, base.ConnParameters{}, err
	}
	for k, v := range requestHeader {
		req.Header[k] = v
	}
	supportBinary := req.URL.Query().Get("b64") == ""
	if supportBinary {
		req.Header.Set("Content-Type", "application/octet-stream")
	} else {
		req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	}

	ret := &clientConn{
		Payload:    payload.New(supportBinary),
		httpClient: client,
		retry:      retry,
		request:    *req,
	}

	go ret.getOpen()

	_, pt, r, err := ret.NextReader()
	if err != nil {
		return nil, base.ConnParameters{}, err
	}
	if pt != base.OPEN {
		return nil, base.ConnParameters{}, errors.New("invalid open")
	}
	conn, err := base.ReadConnParameters(r)
	if err != nil {
		return nil, base.ConnParameters{}, err
	}
	err = r.Close()
	if err != nil {
		return nil, base.ConnParameters{}, err
	}
	query := ret.request.URL.Query()
	query.Set("sid", conn.SID)
	ret.request.URL.RawQuery = query.Encode()

	go ret.serveGet()
	go ret.servePost()

	return ret, conn, nil
}

func (c *clientConn) URL() string {
	return c.request.URL.String()
}

func (c *clientConn) LocalAddr() string {
	return ""
}

func (c *clientConn) RemoteAddr() string {
	return c.request.Host
}

func (c *clientConn) RemoteHeader() http.Header {
	return c.remoteHeader
}

func (c *clientConn) Resume() {
	c.Payload.Resume()
	go c.serveGet()
	go c.servePost()
}

func (c *clientConn) servePost() {
	var buf bytes.Buffer
	req := c.request
	req.Method = "POST"
	req.Body = ioutil.NopCloser(&buf)
	for {
		buf.Reset()
		if err := c.Payload.FlushOut(&buf); err != nil {
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
			c.Payload.Store("post", err)
			c.Close()
			return
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			c.Payload.Store("post", fmt.Errorf("invalid response: %s(%d)", resp.Status, resp.StatusCode))
			c.Close()
			return
		}
	}
}

func (c *clientConn) getOpen() {
	req := c.request
	req.Method = "GET"
	var resp *http.Response
	var err error
	for i := 0; i < c.retry; i++ {
		resp, err = c.httpClient.Do(&req)
		if err == nil {
			break
		}
	}
	if err != nil {
		c.Payload.Store("get", err)
		c.Close()
		return
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("invalid request: %s(%d)", resp.Status, resp.StatusCode)
	}
	var supportBinary bool
	if err == nil {
		mime := resp.Header.Get("Content-Type")
		supportBinary, err = mimeSupportBinary(mime)
	}
	if err != nil {
		c.Payload.Store("get", err)
		c.Close()
		return
	}
	c.remoteHeader = resp.Header
	if err = c.Payload.FeedIn(resp.Body, supportBinary); err != nil {
		return
	}
}

func (c *clientConn) serveGet() {
	req := c.request
	req.Method = "GET"
	for {
		var resp *http.Response
		var err error
		for i := 0; i < c.retry; i++ {
			resp, err = c.httpClient.Do(&req)
			if err == nil {
				break
			}
		}
		if err != nil {
			c.Payload.Store("get", err)
			c.Close()
			return
		}
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("invalid request: %s(%d)", resp.Status, resp.StatusCode)
		}
		var supportBinary bool
		if err == nil {
			mime := resp.Header.Get("Content-Type")
			supportBinary, err = mimeSupportBinary(mime)
		}
		if err != nil {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
			c.Payload.Store("get", err)
			c.Close()
			return
		}
		if err = c.Payload.FeedIn(resp.Body, supportBinary); err != nil {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
			return
		}
	}
}
