package polling

import (
	"io"
	"net/http"
	"time"

	"github.com/googollee/go-socket.io/engineio/transport"
)

const (
	separator    = 0x1e
	binaryPrefix = 'b'
)

func init() {
	transport.Register(transport.Polling, newPolling)
}

type Polling struct {
	callbacks transport.Callbacks
}

func newPolling(pingInterval time.Duration, callbacks transport.Callbacks) transport.Transport {
	return nil
}

func (p *Polling) Name() string {
	return string(transport.Polling)
}

func (p *Polling) Close() error {
	return nil
}

func (p *Polling) SendFrame() (io.WriteCloser, error) {
	return nil, nil
}

func (p *Polling) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
