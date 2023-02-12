package engineio

import (
	"net/http"
	"time"

	"github.com/googollee/go-socket.io/engineio/session"
	"github.com/googollee/go-socket.io/engineio/transport"
)

// CheckerFunc is function to check request.
type CheckerFunc func(*http.Request) (http.Header, error)

// ConnInitorFunc is function to do after create connection.
type ConnInitorFunc func(*http.Request, Conn)

type OptionFunc func(o *Options)

type Options struct {
	PingTimeout  time.Duration
	PingInterval time.Duration

	Transports       []transport.Type
	SessionGenerator session.Generator

	RequestChecker CheckerFunc
	ConnInitor     ConnInitorFunc
}

func newDefaultOptions() *Options {
	return &Options{
		PingTimeout:  time.Minute,
		PingInterval: time.Second * 20,
		Transports: []transport.Type{
			transport.Polling,
			transport.Websocket,
		},
		SessionGenerator: session.NewSessionGenerator(),
		RequestChecker:   defaultChecker,
		ConnInitor:       defaultInitor,
	}
}

func defaultChecker(*http.Request) (http.Header, error) {
	return nil, nil
}

func defaultInitor(*http.Request, Conn) {}
