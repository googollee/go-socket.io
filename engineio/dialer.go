package engineio

import (
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/googollee/go-socket.io/engineio/packet"
	"github.com/googollee/go-socket.io/engineio/transport"
)

// Dialer is dialer configure.
type Dialer struct {
	Transports []transport.Transport
}

// Dial returns a connection which dials to url with requestHeader.
func (d *Dialer) Dial(urlStr string, requestHeader http.Header) (Conn, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	query := u.Query()
	query.Set("EIO", "3")
	u.RawQuery = query.Encode()

	var conn transport.Conn

	for i := len(d.Transports) - 1; i >= 0; i-- {
		if conn != nil {
			conn.Close()
		}

		t := d.Transports[i]

		conn, err = t.Dial(u, requestHeader)
		if err != nil {
			continue
		}

		var params transport.ConnParameters
		if p, ok := conn.(Opener); ok {
			params, err = p.Open()
			if err != nil {
				continue
			}
		} else {
			var pt packet.PacketType
			var r io.ReadCloser
			_, pt, r, err = conn.NextReader()
			if err != nil {
				continue
			}
			func() {
				defer r.Close()
				if pt != packet.OPEN {
					err = errors.New("invalid open")
					return
				}
				params, err = transport.ReadConnParameters(r)
				if err != nil {
					return
				}
			}()
		}
		if err != nil {
			continue
		}
		ret := &client{
			conn:      conn,
			params:    params,
			transport: t.Name(),
			close:     make(chan struct{}),
		}

		go ret.serve()

		return ret, nil
	}

	return nil, err
}
