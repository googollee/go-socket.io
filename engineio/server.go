package engineio

import (
	websocket2 "github.com/gorilla/websocket"
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

func defaultInitor(*http.Request, Conn) {}

// Options is options to create a server.
type Options struct {
	RequestChecker     func(*http.Request) (http.Header, error)
	ConnInitor         func(*http.Request, Conn)
	PingTimeout        time.Duration
	PingInterval       time.Duration
	Transports         []transport.Transport
	SessionIDGenerator SessionIDGenerator
}

func (c *Options) getRequestChecker() func(*http.Request) (http.Header, error) {
	if c != nil && c.RequestChecker != nil {
		return c.RequestChecker
	}
	return defaultChecker
}

func (c *Options) getConnInitor() func(*http.Request, Conn) {
	if c != nil && c.ConnInitor != nil {
		return c.ConnInitor
	}
	return defaultInitor
}

func (c *Options) getPingTimeout() time.Duration {
	if c != nil && c.PingTimeout != 0 {
		return c.PingTimeout
	}
	return time.Minute
}

func (c *Options) getPingInterval() time.Duration {
	if c != nil && c.PingInterval != 0 {
		return c.PingInterval
	}
	return time.Second * 20
}

func (c *Options) getTransport() []transport.Transport {
	if c != nil && len(c.Transports) != 0 {
		return c.Transports
	}
	return []transport.Transport{
		polling.Default,
		websocket.Default,
	}
}

func (c *Options) getSessionIDGenerator() SessionIDGenerator {
	if c != nil && c.SessionIDGenerator != nil {
		return c.SessionIDGenerator
	}
	return &defaultIDGenerator{}
}

// Server is server.
type Server struct {
	transports     *transport.Manager
	pingInterval   time.Duration
	pingTimeout    time.Duration
	sessions       *manager
	requestChecker func(*http.Request) (http.Header, error)
	connInitor     func(*http.Request, Conn)
	connChan       chan Conn
	closeOnce      sync.Once
}

// NewServer returns a server.
func NewServer(opts *Options) (*Server, error) {
	t := transport.NewManager(opts.getTransport())
	return &Server{
		transports:     t,
		pingInterval:   opts.getPingInterval(),
		pingTimeout:    opts.getPingTimeout(),
		requestChecker: opts.getRequestChecker(),
		connInitor:     opts.getConnInitor(),
		sessions:       newManager(opts.getSessionIDGenerator()),
		connChan:       make(chan Conn, 1),
	}, nil
}

// Close closes server.
func (s *Server) Close() error {
	s.closeOnce.Do(func() {
		close(s.connChan)
	})
	return nil
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
	header, err := s.requestChecker(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	for k, v := range header {
		w.Header()[k] = v
	}
	if session == nil {
		if sid != "" {
			http.Error(w, "invalid sid", http.StatusBadRequest)
			return
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
		s.connInitor(r, session)

		go func() {
			w, err := session.nextWriter(base.FrameString, base.OPEN)
			if err != nil {
				session.Close()
				return
			}
			if _, err := session.params.WriteTo(w); err != nil {
				w.Close()
				session.Close()
				return
			}
			if err := w.Close(); err != nil {
				session.Close()
				return
			}
			s.connChan <- session
		}()
	}
	if session.Transport() != t {
		conn, err := tspt.Accept(w, r)
		if err != nil {
			// don't call http.Error() for HandshakeErrors because
			// they get handled by the websocket library internally.
			if _, ok := err.(websocket2.HandshakeError); !ok {
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
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
