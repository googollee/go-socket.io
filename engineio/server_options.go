package engineio

import (
	"net/http"
	"time"

	"github.com/googollee/go-socket.io/engineio/session"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"

	"github.com/googollee/go-socket.io/engineio/transport"
)

// Options is options to create a server.
type Options struct {
	PingTimeout  time.Duration
	PingInterval time.Duration

	Transports         []transport.Transport
	SessionIDGenerator session.IDGenerator

	RequestChecker CheckerFunc
	ConnInitor     ConnInitorFunc
}

type Option func(*Options)

func WithPingTimeout(pingTimeOut time.Duration) Option {
	return func(o *Options) {
		o.PingTimeout = pingTimeOut
	}
}

func WithPingInterval(pingInterval time.Duration) Option {
	return func(o *Options) {
		o.PingInterval = pingInterval
	}
}

func WithTransports(transports []transport.Transport) Option {
	return func(o *Options) {
		o.Transports = transports
	}
}

func WithSessionIDGenerator(SessionIDGenerator session.IDGenerator) Option {
	return func(o *Options) {
		o.SessionIDGenerator = SessionIDGenerator
	}
}

func WithRequestChecker(checkFunc CheckerFunc) Option {
	return func(o *Options) {
		o.RequestChecker = checkFunc
	}
}

func WithConnInitor(connInitFunc ConnInitorFunc) Option {
	return func(o *Options) {
		o.ConnInitor = connInitFunc
	}
}

func GetOptions(opts ...Option) *Options {
	defaultOption := Default()
	// override default opts,user option first
	for _, o := range opts {
		o(defaultOption)
	}
	return defaultOption
}

func Default() *Options {
	return &Options{
		PingTimeout:  20 * time.Second,
		PingInterval: time.Minute,
		Transports: []transport.Transport{
			polling.Default,
			websocket.Default,
		},
		SessionIDGenerator: &session.DefaultIDGenerator{},
		RequestChecker:     defaultChecker,
		ConnInitor:         defaultInitor,
	}
}

func defaultChecker(*http.Request) (http.Header, error) {
	return nil, nil
}

func defaultInitor(*http.Request, Conn) {}
