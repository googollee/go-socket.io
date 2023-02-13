package polling

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/googollee/go-socket.io/engineio/packet"
	"github.com/googollee/go-socket.io/engineio/payload"
	"github.com/googollee/go-socket.io/engineio/protocol"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/utils"
)

type Connection struct {
	*payload.Payload

	client *http.Client

	request      http.Request
	remoteHeader atomic.Value
}

func New() *Connection {
	return &Connection{
		client: &http.Client{},
	}
}

func (c *Connection) Do(req *http.Request) (*Connection, error) {
	if req == nil {
		return nil, errors.New("")
	}

	req.URL.Query().Set(protocol.Transport, transport.Polling)

	return c.dial(req)
}

func (c *Connection) dial(req *http.Request) (*Connection, error) {
	req, err := http.NewRequest(http.MethodGet, req.URL.String(), nil)
	if err != nil {
		return nil, err
	}

	for k, v := range req.Header {
		req.Header[k] = v
	}

	supportBinary := req.URL.Query().Get(protocol.Base64Encoding) == ""
	if supportBinary {
		req.Header.Set("Content-Type", "application/octet-stream")
	} else {
		req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	}

	return &Connection{
		Payload: payload.New(supportBinary),
		client:  c.client,
		request: *req,
	}, nil
}

func (c *Connection) Open() (transport.ConnParameters, error) {
	go c.getOpen()

	_, pt, r, err := c.Payload.NextReader()
	if err != nil {
		return transport.ConnParameters{}, err
	}

	if pt != packet.OPEN {
		r.Close()
		return transport.ConnParameters{}, errors.New("invalid open")
	}

	conn, err := transport.ReadConnParameters(r)
	if err != nil {
		r.Close()
		return transport.ConnParameters{}, err
	}

	err = r.Close()

	if err != nil {
		return transport.ConnParameters{}, err
	}

	query := c.request.URL.Query()
	query.Set(protocol.SID, conn.SID)
	c.request.URL.RawQuery = query.Encode()

	go c.serveGet()
	go c.servePost()

	return conn, nil
}

func (c *Connection) URL() url.URL {
	return *c.request.URL
}

func (c *Connection) LocalAddr() net.Addr {
	return Addr{""}
}

func (c *Connection) RemoteAddr() net.Addr {
	return Addr{c.request.Host}
}

func (c *Connection) RemoteHeader() http.Header {
	ret := c.remoteHeader.Load()
	if ret == nil {
		return nil
	}
	return ret.(http.Header)
}

func (c *Connection) Resume() {
	c.Payload.Resume()

	go c.serveGet()
	go c.servePost()
}

func (c *Connection) servePost() {
	req := c.request
	reqUrl := *req.URL

	req.URL = &reqUrl
	req.Method = http.MethodPost

	var buf bytes.Buffer
	req.Body = ioutil.NopCloser(&buf)

	query := reqUrl.Query()
	for {
		buf.Reset()

		if err := c.Payload.FlushOut(&buf); err != nil {
			return
		}
		query.Set(protocol.TimestampTag, utils.Timestamp())
		req.URL.RawQuery = query.Encode()

		resp, err := c.httpClient.Do(&req)
		if err != nil {
			if err = c.Payload.Store(http.MethodPost, err); err != nil {
				log.Println("store post error", err)
			}

			c.Payload.Close()

			return
		}

		discardBody(resp.Body)

		if resp.StatusCode != http.StatusOK {
			err = c.Payload.Store(http.MethodPost, fmt.Errorf("invalid response: %s(%d)", resp.Status, resp.StatusCode))
			if err != nil {
				log.Println("store post error", err)
			}

			c.Payload.Close()

			return
		}

		c.remoteHeader.Store(resp.Header)
	}
}

func (c *Connection) getOpen() {
	req := c.request
	query := req.URL.Query()

	reqUrl := *req.URL
	req.URL = &reqUrl
	req.Method = http.MethodGet

	query.Set(protocol.TimestampTag, utils.Timestamp())
	req.URL.RawQuery = query.Encode()

	resp, err := c.client.Do(&req)
	if err != nil {
		if err = c.Payload.Store(http.MethodGet, err); err != nil {
			log.Println("store get error", err)
		}

		c.Payload.Close()
		return
	}

	defer func() {
		discardBody(resp.Body)
	}()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("invalid request: %s(%d)", resp.Status, resp.StatusCode)
	}

	var isSupportBinary bool
	if err == nil {
		mime := resp.Header.Get("Content-Type")
		isSupportBinary, err = utils.MimeIsSupportBinary(mime)
		if err != nil {
			log.Println("check mime support binary", err)
		}
	}

	if err != nil {
		if err = c.Payload.Store(http.MethodGet, err); err != nil {
			log.Println("store get error", err)
		}
		c.Payload.Close()

		return
	}

	c.remoteHeader.Store(resp.Header)

	if err = c.Payload.FeedIn(resp.Body, isSupportBinary); err != nil {
		return
	}
}

func (c *Connection) serveGet() {
	req := c.request
	reqUrl := *req.URL

	req.URL = &reqUrl
	req.Method = http.MethodGet

	query := req.URL.Query()
	for {
		query.Set(protocol.TimestampTag, utils.Timestamp())
		req.URL.RawQuery = query.Encode()

		resp, err := c.httpClient.Do(&req)
		if err != nil {
			if err = c.Payload.Store(http.MethodGet, err); err != nil {
				log.Println("store get error", err)
			}
			c.Close()

			return
		}

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("invalid request: %s(%d)", resp.Status, resp.StatusCode)
		}

		var isSupportBinary bool
		if err == nil {
			mime := resp.Header.Get("Content-Type")
			isSupportBinary, err = utils.MimeIsSupportBinary(mime)
			if err != nil {
				log.Println("check mime support binary", err)
			}
		}

		if err != nil {
			discardBody(resp.Body)

			if err = c.Payload.Store(http.MethodGet, err); err != nil {
				log.Println("store get error", err)
			}

			c.Close()

			return
		}

		if err = c.Payload.FeedIn(resp.Body, isSupportBinary); err != nil {
			discardBody(resp.Body)

			return
		}

		c.remoteHeader.Store(resp.Header)
	}
}

func discardBody(body io.ReadCloser) {
	_, err := io.Copy(ioutil.Discard, body)
	if err != nil {
		log.Println("copy from body resp to discard", err)
	}
	body.Close()
}
