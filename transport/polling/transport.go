package polling

import (
	"net/http"
	"time"

	"github.com/googollee/go-engine.io/base"
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

// Open gets connection parameters from url.
func (t *Transport) Dial(url string, requestHeader http.Header) (base.Conn, error) {
	return dial(t.Retry, t.Client, url, requestHeader)
}
