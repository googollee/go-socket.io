package engineio

import (
	"io"
	"net/http"
	"sync"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/transport"
)

type Server struct {
	transports     *transport.Manager
	sessions       *manager
	requestChecker func(*http.Request) (base.ConnParameters, http.Header, error)
	locker         sync.RWMutex
	connChan       chan Conn
}

func NewServer(transports []transport.Transport) *Server {
	t := transport.NewManager(transports)
	return &Server{
		transports: t,
		sessions:   newManager(),
		connChan:   make(chan Conn, 1),
	}
}

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
	t := query.Get("t")
	transport := s.transports.Get(t)
	if transport == nil {
		http.Error(w, "invalid transport", http.StatusBadRequest)
		return
	}
	if session == nil {
		if sid != "" {
			http.Error(w, "invalid sid", http.StatusBadRequest)
			return
		}
		params, header, err := s.requestChecker(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		for k, v := range header {
			w.Header()[k] = v
		}
		conn, err := transport.Accept(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		session = newSession(s.sessions, t, conn, params)
		s.connChan <- session
	}
	if session.Transport() != t {
		params, header, err := s.requestChecker(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		for k, v := range header {
			w.Header()[k] = v
		}
		conn, err := transport.Accept(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		session.upgrade(params, t, conn)
		if handler, ok := conn.(http.Handler); ok {
			handler.ServeHTTP(w, r)
		}
		return
	}
	session.serveHTTP(w, r)
}
