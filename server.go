package socketio

import (
	"net/http"

	engineio "github.com/googollee/go-engine.io"
)

// Server is a go-socket.io server.
type Server struct {
	broadcast Broadcast
	handlers  map[string]*namespaceHandler
	eio       *engineio.Server
}

// NewServer returns a server.
func NewServer(c *engineio.Options) (*Server, error) {
	eio, err := engineio.NewServer(c)
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
func (s *Server) OnError(nsp string, f func(error)) {
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
		conn, err := s.eio.Accept()
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

// RoomLen gives number of connections in the room
func (s *Server) RoomLen(room string) int {
	return s.broadcast.Len(room)
}

//  Rooms gives list of all the rooms
func (s *Server) Rooms() []string {
	return s.broadcast.Rooms(nil)
}

func (s *Server) serveConn(c engineio.Conn) {
	_, err := newConn(c, s.handlers, s.broadcast)
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
