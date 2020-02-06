package polling

import (
	"net/http"
	"net/url"
	"time"

	"github.com/googollee/go-socket.io/connection/base"
)

// Transport is the transport of polling.
type Transport struct {
	Client *http.Client
}

// Default is the default transport.
var Default = &Transport{
	Client: &http.Client{
		Timeout: time.Minute,
	},
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

// Dial dials connection to url.
func (t *Transport) Dial(u *url.URL, requestHeader http.Header) (base.Conn, error) {
	query := u.Query()
	query.Set("transport", t.Name())
	u.RawQuery = query.Encode()
	return dial(t.Client, u, requestHeader)
}
