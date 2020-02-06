package socketio

import (
	"github.com/googollee/go-socket.io/base"
	"github.com/googollee/go-socket.io/transport"
	"github.com/googollee/go-socket.io/transport/polling"
	"github.com/googollee/go-socket.io/transport/websocket"
	"io"
	"net/http"
	"sync"
	"time"
)

func defaultChecker(*http.Request) (http.Header, error) {
	return nil, nil
}

func defaultInitor(*http.Request, base.Conn) {}

// Options is options to create a server.
type Options struct {
	RequestChecker     func(*http.Request) (http.Header, error)
	ConnInitor         func(*http.Request, base.Conn)
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

func (c *Options) getConnInitor() func(*http.Request, base.Conn) {
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
	closeOnce sync.Once

	transports     *transport.Manager
	pingInterval   time.Duration
	pingTimeout    time.Duration
	sessions       *manager
	broadcast      Broadcast
	handlers       map[string]*namespaceHandler
	requestChecker func(*http.Request) (http.Header, error)
	connInitor     func(*http.Request, base.Conn)
	connChan       chan base.Conn
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
		connChan:       make(chan base.Conn, 1),
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
func (s *Server) Accept() (base.Conn, error) {
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

		//FIXME: need refactor code
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

// OnConnect set a handler function f to handle open event for
// namespace nsp.
func (s *Server) OnConnect(nsp string, f func(base.Conn) error) {
	h := s.getNamespace(nsp)
	h.OnConnect(f)
}

// OnDisconnect set a handler function f to handle disconnect event for
// namespace nsp.
func (s *Server) OnDisconnect(nsp string, f func(base.Conn, string)) {
	h := s.getNamespace(nsp)
	h.OnDisconnect(f)
}

// OnError set a handler function f to handle error for namespace nsp.
func (s *Server) OnError(nsp string, f func(base.Conn, error)) {
	h := s.getNamespace(nsp)
	h.OnError(f)
}

// OnEvent set a handler function f to handle event for namespace nsp.
func (s *Server) OnEvent(nsp, event string, f interface{}) {
	h := s.getNamespace(nsp)
	h.OnEvent(event, f)
}

// Serve serves go-socket.io server
func (s *Server) Serve() error {
	for {
		conn, err := s.Accept()
		if err != nil {
			return err
		}
		go s.serveConn(conn)
	}
}

// JoinRoom joins given connection to the room
func (s *Server) JoinRoom(room string, connection base.Conn) {
	s.broadcast.Join(room, connection)
}

// LeaveRoom leaves given connection from the room
func (s *Server) LeaveRoom(room string, connection base.Conn) {
	s.broadcast.Leave(room, connection)
}

// LeaveAllRooms leaves the given connection from all rooms
func (s *Server) LeaveAllRooms(connection base.Conn) {
	s.broadcast.LeaveAll(connection)
}

// ClearRoom clears the room
func (s *Server) ClearRoom(room string) {
	s.broadcast.Clear(room)
}

// BroadcastToRoom broadcasts given event & args to all the connections in the room
func (s *Server) BroadcastToRoom(room, event string, args ...interface{}) {
	s.broadcast.Send(room, event, args...)
}

// Emit emit to message given connectId, event & args to target connetion
func (s *Server) Emit(connectID, event string, args ...interface{}) {
	s.broadcast.Emit(connectID, event, args...)
}

// RoomLen gives number of connections in the room
func (s *Server) RoomLen(room string) int {
	return s.broadcast.Len(room)
}

// Rooms gives list of all the rooms
func (s *Server) Rooms() []string {
	return s.broadcast.Rooms(nil)
}

func (s *Server) serveConn(c base.Conn) {
	_, err := newConn(c, s.handlers, s.broadcast)
	if err != nil {
		root := s.handlers[""]
		if root != nil && root.onError != nil {
			root.onError(nil, err)
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
