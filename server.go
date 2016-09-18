package engineio

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/transport"
	"github.com/googollee/go-engine.io/transport/polling"
	"github.com/googollee/go-engine.io/transport/websocket"
)

func defaultChecker(*http.Request) (http.Header, error) {
	return nil, nil
}

// Config is configure.
type Config struct {
	RequestChecker func(*http.Request) (http.Header, error)
	PingTimeout    time.Duration
	PingInterval   time.Duration
	Transports     []transport.Transport
}

func (c *Config) fillNil() {
	if c.RequestChecker == nil {
		c.RequestChecker = defaultChecker
	}
	if c.PingTimeout == 0 {
		c.PingTimeout = time.Minute
	}
	if c.PingInterval == 0 {
		c.PingInterval = time.Second * 20
	}
	if len(c.Transports) == 0 {
		c.Transports = []transport.Transport{
			polling.Default,
			websocket.Default,
		}
	}
}

// Server is server.
type Server struct {
	transports     *transport.Manager
	pingInterval   time.Duration
	pingTimeout    time.Duration
	sessions       *manager
	requestChecker func(*http.Request) (http.Header, error)
	locker         sync.RWMutex
	connChan       chan Conn
}

// NewServer returns a server.
func NewServer(c *Config) (*Server, error) {
	if c == nil {
		c = &Config{}
	}
	conf := *c
	conf.fillNil()
	t := transport.NewManager(conf.Transports)
	return &Server{
		transports:     t,
		pingInterval:   conf.PingInterval,
		pingTimeout:    conf.PingTimeout,
		requestChecker: conf.RequestChecker,
		sessions:       newManager(),
		connChan:       make(chan Conn, 1),
	}, nil
}

// Accept accepts a connection.
func (s *Server) Accept() (Conn, error) {
	c := <-s.connChan
	if c == nil {
		return nil, io.EOF
	}
	return c, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	sid := query.Get("sid")
	session := s.sessions.Get(sid)
	t := query.Get("transport")
	tspt := s.transports.Get(t)

	if tspt == nil {
		http.Error(w, "invalid transport", http.StatusBadRequest)
		return
	}
	if session == nil {
		if sid != "" {
			http.Error(w, "invalid sid", http.StatusBadRequest)
			return
		}
		header, err := s.requestChecker(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		for k, v := range header {
			w.Header()[k] = v
		}
		conn, err := tspt.Accept(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		params := base.ConnParameters{
			PingInterval: s.pingInterval,
			PingTimeout:  s.pingTimeout,
			Upgrades:     s.transports.UpgradeFrom(t),
		}
		session, err = newSession(s.sessions, t, conn, params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		s.connChan <- session
	}
	if session.Transport() != t {
		header, err := s.requestChecker(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		for k, v := range header {
			w.Header()[k] = v
		}
		conn, err := tspt.Accept(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		session.upgrade(t, conn)
		if handler, ok := conn.(http.Handler); ok {
			handler.ServeHTTP(w, r)
		}
		return
	}
	session.serveHTTP(w, r)
}
