package socketio

import (
	"github.com/googollee/go-engine.io"
	"net/http"
	"time"
)

// Server is the server of socket.io.
type Server struct {
	*namespace
	broadcast BroadcastAdaptor
	eio       *engineio.Server
}

// NewServer returns the server supported given transports. If transports is nil, the server will use ["polling", "websocket"] as default.
func NewServer(transportNames []string) (*Server, error) {
	eio, err := engineio.NewServer(transportNames)
	if err != nil {
		return nil, err
	}
	ret := &Server{
		namespace: newNamespace(newBroadcastDefault()),
		eio:       eio,
	}
	go ret.loop()
	return ret, nil
}

// SetPingTimeout sets the timeout of a connection ping. When it times out, the server will close the connection with the client. Default is 60s.
func (s *Server) SetPingTimeout(t time.Duration) {
	s.eio.SetPingTimeout(t)
}

// SetPingInterval sets the interval of pings. Default is 25s.
func (s *Server) SetPingInterval(t time.Duration) {
	s.eio.SetPingInterval(t)
}

// SetMaxConnection sets the maximum number of connections with clients. Default is 1000.
func (s *Server) SetMaxConnection(n int) {
	s.eio.SetMaxConnection(n)
}

// GetMaxConnection returns the current max connection
func (s *Server) GetMaxConnection() int {
	return s.eio.GetMaxConnection()
}

// Count returns the current number of connected clients in session
func (s *Server) Count() int {
	return s.eio.Count()
}

// SetAllowRequest sets the middleware function when a connection is established. If a non-nil value is returned, the connection won't be established. Default will allow all connections.
func (s *Server) SetAllowRequest(f func(*http.Request) error) {
	s.eio.SetAllowRequest(f)
}

// SetAllowUpgrades sets whether server allows transport upgrades. Default is true.
func (s *Server) SetAllowUpgrades(allow bool) {
	s.eio.SetAllowUpgrades(allow)
}

// SetCookie sets the name of the cookie used by engine.io. Default is "io".
func (s *Server) SetCookie(prefix string) {
	s.eio.SetCookie(prefix)
}

// SetNewId sets the callback func to generate new connection id. By default, id is generated from remote address + current time stamp
func (s *Server) SetNewId(f func(*http.Request) string) {
	s.eio.SetNewId(f)
}

// SetSessionsManager sets the sessions as server's session manager. Default sessions is a single process manager. You can customize it as a load balancer.
func (s *Server) SetSessionManager(sessions engineio.Sessions) {
	s.eio.SetSessionManager(sessions)
}

// SetAdaptor sets the adaptor of broadcast. Default is an in-process broadcast implementation.
func (s *Server) SetAdaptor(adaptor BroadcastAdaptor) {
	s.namespace = newNamespace(adaptor)
}

// ServeHTTP handles http requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.eio.ServeHTTP(w, r)
}

// BroadcastTo is a server level broadcast function.
func (s *Server) BroadcastTo(room, message string, args ...interface{}) {
	s.namespace.BroadcastTo(room, message, args...)
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
