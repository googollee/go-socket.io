package polling

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/googollee/go-socket.io/connection/base"
	"github.com/googollee/go-socket.io/connection/payload"
)

type clientConn struct {
	*payload.Payload

	httpClient   *http.Client
	request      http.Request
	remoteHeader atomic.Value
}

func dial(client *http.Client, url *url.URL, requestHeader http.Header) (*clientConn, error) {
	if client == nil {
		client = &http.Client{}
	}
	req, err := http.NewRequest("", url.String(), nil)
	if err != nil {
		return nil, err
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
		request:    *req,
	}
	return ret, nil
}

func (c *clientConn) Open() (base.ConnParameters, error) {
	go c.getOpen()

	_, pt, r, err := c.NextReader()
	if err != nil {
		return base.ConnParameters{}, err
	}
	if pt != base.OPEN {
		r.Close()
		return base.ConnParameters{}, errors.New("invalid open")
	}
	conn, err := base.ReadConnParameters(r)
	if err != nil {
		r.Close()
		return base.ConnParameters{}, err
	}
	err = r.Close()
	if err != nil {
		return base.ConnParameters{}, err
	}
	query := c.request.URL.Query()
	query.Set("sid", conn.SID)
	c.request.URL.RawQuery = query.Encode()

	go c.serveGet()
	go c.servePost()

	return conn, nil
}

func (c *clientConn) URL() url.URL {
	return *c.request.URL
}

func (c *clientConn) LocalAddr() net.Addr {
	return Addr{""}
}

func (c *clientConn) RemoteAddr() net.Addr {
	return Addr{c.request.Host}
}

func (c *clientConn) RemoteHeader() http.Header {
	ret := c.remoteHeader.Load()
	if ret == nil {
		return nil
	}
	return ret.(http.Header)
}

func (c *clientConn) Resume() {
	c.Payload.Resume()
	go c.serveGet()
	go c.servePost()
}

func (c *clientConn) servePost() {
	var buf bytes.Buffer
	req := c.request
	url := *req.URL
	req.URL = &url
	query := url.Query()
	req.Method = "POST"
	req.Body = ioutil.NopCloser(&buf)
	for {
		buf.Reset()
		if err := c.Payload.FlushOut(&buf); err != nil {
			return
		}
		query.Set("t", base.Timestamp())
		req.URL.RawQuery = query.Encode()
		resp, err := c.httpClient.Do(&req)
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
		c.remoteHeader.Store(resp.Header)
	}
}

func (c *clientConn) getOpen() {
	req := c.request
	query := req.URL.Query()
	url := *req.URL
	req.URL = &url
	req.Method = "GET"
	query.Set("t", base.Timestamp())
	req.URL.RawQuery = query.Encode()
	resp, err := c.httpClient.Do(&req)
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
	c.remoteHeader.Store(resp.Header)
	if err = c.Payload.FeedIn(resp.Body, supportBinary); err != nil {
		return
	}
}

func (c *clientConn) serveGet() {
	req := c.request
	query := req.URL.Query()
	url := *req.URL
	req.URL = &url
	req.Method = "GET"
	for {
		query.Set("t", base.Timestamp())
		req.URL.RawQuery = query.Encode()
		resp, err := c.httpClient.Do(&req)
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
		c.remoteHeader.Store(resp.Header)
	}
}
