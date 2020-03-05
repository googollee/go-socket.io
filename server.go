package socketio

import (
	engineio "github.com/googollee/go-socket.io/connection"
	"github.com/googollee/go-socket.io/connection/transport"
	"net/http"
)

// Server is server.
type Server struct {
	transports *transport.Manager
	broadcast  Broadcast
	handlers   map[string]*namespaceHandler
	eio        *engineio.Server
}

// NewServer returns a server.
func NewServer(opts *engineio.Options) (*Server, error) {
	eio, err := engineio.NewServer(opts)
	if err != nil {
		return nil, err
	}
	return &Server{
		handlers:  make(map[string]*namespaceHandler),
		eio:       eio,
		broadcast: NewBroadcast(),
	}, nil
}

// Close closes server.
func (s *Server) Close() error {
	return s.eio.Close()
}

// Accept accepts a connection.
func (s *Server) Accept() (engineio.Conn, error) {
	return s.eio.Accept()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.eio.ServeHTTP(w, r)
}

// OnConnect set a handler function f to handle open event for
// namespace nsp.
func (s *Server) OnConnect(nsp string, f func(Conn) error) {
	h := s.getNamespace(nsp)
	h.OnConnect(f)
}

// OnDisconnect set a handler function f to handle disconnect event for
// namespace nsp.
func (s *Server) OnDisconnect(nsp string, f func(Conn, string)) {
	h := s.getNamespace(nsp)
	h.OnDisconnect(f)
}

// OnError set a handler function f to handle error for namespace nsp.
func (s *Server) OnError(nsp string, f func(Conn, error)) {
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
func (s *Server) JoinRoom(room string, connection Conn) {
	s.broadcast.Join(room, connection)
}

// LeaveRoom leaves given connection from the room
func (s *Server) LeaveRoom(room string, connection Conn) {
	s.broadcast.Leave(room, connection)
}

// LeaveAllRooms leaves the given connection from all rooms
func (s *Server) LeaveAllRooms(connection Conn) {
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

// Emit emit to message given connectId, event & args to target connection
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

func (s *Server) serveConn(c engineio.Conn) {
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
