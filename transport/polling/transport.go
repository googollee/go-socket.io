package polling

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/payload"
)

// Transport is the transport of polling.
type Transport struct {
	Client *http.Client
	Retry  int
}

// Default is the default transport.
var Default = &Transport{
	Client: &http.Client{
		Timeout: time.Minute,
	},
	Retry: 3,
}

// Name is the name of transport.
func (t *Transport) Name() string {
	return "polling"
}

// Accept accepts a http request and create Conn.
func (t *Transport) Accept(w http.ResponseWriter, r *http.Request) (base.Conn, error) {
	conn := newServerConn(r)
	return conn, nil
}

// Dial dials to url with requestHeader and returns connection.
func (t *Transport) Dial(url string, requestHeader http.Header) (base.Conn, error) {
	ret, err := t.dial(url, requestHeader)
	if err != nil {
		return nil, err
	}

	go ret.doGet(false)
	go ret.doPost()

	return ret, nil
}

// Open gets connection parameters from url.
func (t *Transport) Open(url string, requestHeader http.Header) (base.ConnParameters, error) {
	c, err := t.dial(url, requestHeader)
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
	ret, err := base.ReadConnParameters(r)
	if err != nil {
		return base.ConnParameters{}, base.OpErr(url, "open", err)
	}
	t.Client.Timeout = ret.PingTimeout
	return ret, nil
}

func (t *Transport) dial(url string, requestHeader http.Header) (*clientConn, error) {
	client := t.Client
	if client == nil {
		client = &http.Client{}
	}
	if t.Retry == 0 {
		t.Retry = 3
	}
	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, base.OpErr(url, "create request", err)
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
		supportBinary: supportBinary,
		retry:         t.Retry,
		request:       *req,
		httpClient:    client,
		signal:        payload.NewSignal(),
	}
	ret.signal.StoreError(base.OpErr(url, "i/o", io.EOF))
	ret.encoder = payload.NewEncoder(supportBinary, ret.signal)
	ret.decoder = payload.NewDecoder(ret.signal)

	return ret, nil
}
