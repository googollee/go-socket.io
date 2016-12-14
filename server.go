package socketio

import (
	"net/http"

	"gopkg.in/googollee/go-engine.io.v1"
)

type Server struct {
	handlers map[string]*namespaceHandler
	eio      *engineio.Server
}

func NewServer(c *engineio.Config) (*Server, error) {
	eio, err := engineio.NewServer(c)
	if err != nil {
		return nil, err
	}
	return &Server{
		handlers: make(map[string]*namespaceHandler),
		eio:      eio,
	}, nil
}

func (s *Server) Close() error {
	return s.eio.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.eio.ServeHTTP(w, r)
}

func (s *Server) OnConnect(nsp string, f func(Conn) error) {
	h := s.getNamespace(nsp)
	h.OnConnect(f)
}

func (s *Server) OnDisconnect(nsp string, f func(Conn, string)) {
	h := s.getNamespace(nsp)
	h.OnDisconnect(f)
}

func (s *Server) OnError(nsp string, f func(error)) {
	h := s.getNamespace(nsp)
	h.OnError(f)
}

func (s *Server) OnEvent(nsp, event string, f interface{}) {
	h := s.getNamespace(nsp)
	h.OnEvent(event, f)
}

func (s *Server) Serve() error {
	for {
		conn, err := s.eio.Accept()
		if err != nil {
			return err
		}
		go s.serveConn(conn)
	}
}

func (s *Server) serveConn(c engineio.Conn) {
	_, err := newConn(c, s.handlers)
	if err != nil {
		root := s.handlers[""]
		if root != nil && root.onError != nil {
			root.onError(err)
		}
		return
	}
}

func (s *Server) getNamespace(nsp string) *namespaceHandler {
	if nsp == "/" {
		nsp = ""
	}
	ret, ok := s.handlers[nsp]
	if ok {
		return ret
	}
	handler := newHandler()
	s.handlers[nsp] = handler
	return handler
}
