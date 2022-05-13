package engineio

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/googollee/go-socket.io/engineio/session"
	"github.com/googollee/go-socket.io/engineio/transport"
)

// Server is instance of server
type Server struct {
	pingInterval time.Duration
	pingTimeout  time.Duration

	transports *transport.Manager
	sessions   *session.Manager

	requestChecker CheckerFunc
	connInitor     ConnInitorFunc

	connChan  chan Conn
	closeOnce sync.Once
}

// NewServer returns a server.
func NewServer(opts *Options) *Server {
	return &Server{
		transports:     transport.NewManager(opts.getTransport()),
		pingInterval:   opts.getPingInterval(),
		pingTimeout:    opts.getPingTimeout(),
		requestChecker: opts.getRequestChecker(),
		connInitor:     opts.getConnInitor(),
		sessions:       session.NewManager(opts.getSessionIDGenerator()),
		connChan:       make(chan Conn, 1),
	}
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

func (s *Server) Addr() net.Addr {
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	reqTransport := query.Get("transport")
	srvTransport, ok := s.transports.Get(reqTransport)
	if !ok || srvTransport == nil {
		http.Error(w, fmt.Sprintf("invalid transport: %s", srvTransport), http.StatusBadRequest)
		return
	}

	header, err := s.requestChecker(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("request checker err: %s", err.Error()), http.StatusBadGateway)
		return
	}

	for k, v := range header {
		w.Header()[k] = v
	}

	sid := query.Get("sid")
	reqSession, ok := s.sessions.Get(sid)
	// if we can't find session in current session pool, let's create this. behaviour for new connections
	if !ok || reqSession == nil {
		if sid != "" {
			http.Error(w, fmt.Sprintf("invalid sid value: %s", sid), http.StatusBadRequest)
			return
		}

		transportConn, err := srvTransport.Accept(w, r)
		if err != nil {
			http.Error(w, fmt.Sprintf("transport accept err: %s", err.Error()), http.StatusBadGateway)
			return
		}

		reqSession, err = s.newSession(r.Context(), transportConn, reqTransport)
		if err != nil {
			http.Error(w, fmt.Sprintf("create new session err: %s", err.Error()), http.StatusBadRequest)
			return
		}

		s.connInitor(r, reqSession)
	}

	// try upgrade current connection
	if reqSession.Transport() != reqTransport {
		transportConn, err := srvTransport.Accept(w, r)
		if err != nil {
			// don't call http.Error() for HandshakeErrors because
			// they get handled by the websocket library internally.
			if _, ok := err.(websocket.HandshakeError); !ok {
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
			return
		}

		reqSession.Upgrade(reqTransport, transportConn)

		if handler, ok := transportConn.(http.Handler); ok {
			handler.ServeHTTP(w, r)
		}
		return
	}

	reqSession.ServeHTTP(w, r)
}

// Count counts connected
func (s *Server) Count() int {
	return s.sessions.Count()
}

// Remove session from sessions pool. Experimental API.
func (s *Server) Remove(sid string) {
	s.sessions.Remove(sid)
}

func (s *Server) newSession(_ context.Context, conn transport.Conn, reqTransport string) (*session.Session, error) {
	params := transport.ConnParameters{
		PingInterval: s.pingInterval,
		PingTimeout:  s.pingTimeout,
		Upgrades:     s.transports.UpgradeFrom(reqTransport),
	}

	sid := s.sessions.NewID()
	newSession, err := session.New(conn, sid, reqTransport, params)
	if err != nil {
		return nil, err
	}

	go func(newSession *session.Session) {
		if err = newSession.InitSession(); err != nil {
			log.Println("init new session:", err)

			return
		}

		s.sessions.Add(newSession)

		s.connChan <- newSession
	}(newSession)

	return newSession, nil
}
