package socketio

import (
	"github.com/googollee/go-engine.io"
	"net/http"
	"time"
)

type Config struct {
	PingTimeout   time.Duration
	PingInterval  time.Duration
	AllowRequest  func(*http.Request) error
	Transports    []string
	AllowUpgrades bool
	Cookie        string
	Adaptor       BroadcastAdaptor
}

var DefaultConfig = Config{
	PingTimeout:   60000 * time.Millisecond,
	PingInterval:  25000 * time.Millisecond,
	AllowRequest:  func(*http.Request) error { return nil },
	Transports:    []string{"polling", "websocket"},
	AllowUpgrades: true,
	Cookie:        "io",
	Adaptor:       newBroadcastDefault(),
}

type Server struct {
	*namespace
	eio *engineio.Server
}

func NewServer(conf Config) *Server {
	econf := engineio.Config{
		PingTimeout:   conf.PingTimeout,
		PingInterval:  conf.PingInterval,
		AllowRequest:  conf.AllowRequest,
		Transports:    conf.Transports,
		AllowUpgrades: conf.AllowUpgrades,
		Cookie:        conf.Cookie,
	}
	ret := &Server{
		namespace: newNamespace(conf.Adaptor),
		eio:       engineio.NewServer(econf),
	}
	go ret.loop()
	return ret
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.eio.ServeHTTP(w, r)
}

func (s *Server) loop() {
	for {
		conn, err := s.eio.Accept()
		if err != nil {
			return
		}
		s := newSocket(conn, s.baseHandler)
		go func(s *socket) {
			s.loop()
		}(s)
	}
}
