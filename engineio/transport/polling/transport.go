package polling

import (
	"net/http"
	"net/url"
	"time"

	"github.com/googollee/go-socket.io/engineio/payload"
	"github.com/googollee/go-socket.io/engineio/transport"
)

// Transport is the transport of polling.
type Transport struct {
	Client      *http.Client
	CheckOrigin func(r *http.Request) bool
}

// Default is the default transport.
var Default = &Transport{
	Client: &http.Client{
		Timeout: time.Minute,
	},
	CheckOrigin: nil,
}

// Name is the name of transport.
func (t *Transport) Name() string {
	return "polling"
}

// Accept accepts a http request and create Conn.
func (t *Transport) Accept(w http.ResponseWriter, r *http.Request) (transport.Conn, error) {
	conn := newServerConn(t, r)
	return conn, nil
}

// Dial dials connection to url.
func (t *Transport) Dial(u *url.URL, requestHeader http.Header) (transport.Conn, error) {
	query := u.Query()
	query.Set("transport", t.Name())
	u.RawQuery = query.Encode()

	client := t.Client
	if client == nil {
		client = Default.Client
	}

	return dial(client, u, requestHeader)
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

	return &clientConn{
		Payload:    payload.New(supportBinary),
		httpClient: client,
		request:    *req,
	}, nil
}
