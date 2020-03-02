package socketio

import (
	"net/http"

	engineio "github.com/googollee/go-engine.io"
)

// Server is a go-socket.io server.
type Server struct {
	handlers map[string]*namespaceHandler
	eio      *engineio.Server
}

// NewServer returns a server.
func NewServer(c *engineio.Options) (*Server, error) {
	eio, err := engineio.NewServer(c)
	if err != nil {
		return nil, err
	}
	return &Server{
		handlers: make(map[string]*namespaceHandler),
		eio:      eio,
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
	h := s.getNamespace(nsp, true)
	h.OnConnect(f)
}

// OnDisconnect set a handler function f to handle disconnect event for
// namespace nsp.
func (s *Server) OnDisconnect(nsp string, f func(Conn, string)) {
	h := s.getNamespace(nsp, true)
	h.OnDisconnect(f)
}

// OnError set a handler function f to handle error for namespace nsp.
func (s *Server) OnError(nsp string, f func(Conn, error)) {
	h := s.getNamespace(nsp, true)
	h.OnError(f)
}

// OnEvent set a handler function f to handle event for namespace nsp.
func (s *Server) OnEvent(nsp, event string, f interface{}) {
	h := s.getNamespace(nsp, true)
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
func (s *Server) JoinRoom(namespace string, room string, connection Conn) bool {
	nspHandler := s.getNamespace(namespace, false)
	if nspHandler != nil {
		nspHandler.broadcast.Join(room, connection)
		return true
	}
	return false
}

// LeaveRoom leaves given connection from the room
func (s *Server) LeaveRoom(namespace string, room string, connection Conn) bool {
	nspHandler := s.getNamespace(namespace, false)
	if nspHandler != nil {
		nspHandler.broadcast.Leave(room, connection)
		return true
	}
	return false
}

// LeaveAllRooms leaves the given connection from all rooms
func (s *Server) LeaveAllRooms(namespace string, connection Conn) bool {
	nspHandler := s.getNamespace(namespace, false)
	if nspHandler != nil {
		nspHandler.broadcast.LeaveAll(connection)
		return true
	}
	return false
}

// ClearRoom clears the room
func (s *Server) ClearRoom(namespace string, room string) bool {
	nspHandler := s.getNamespace(namespace, false)
	if nspHandler != nil {
		nspHandler.broadcast.Clear(room)
		return true
	}
	return false
}

// BroadcastToRoom broadcasts given event & args to all the connections in the room
func (s *Server) BroadcastToRoom(namespace string, room, event string, args ...interface{}) bool {
	nspHandler := s.getNamespace(namespace, false)
	if nspHandler != nil {
		nspHandler.broadcast.Send(room, event, args...)
		return true
	}
	return false
}

// RoomLen gives number of connections in the room
func (s *Server) RoomLen(namespace string, room string) int {
	nspHandler := s.getNamespace(namespace, false)
	if nspHandler != nil {
		return nspHandler.broadcast.Len(room)
	}
	return -1
}

// Rooms gives list of all the rooms
func (s *Server) Rooms(namespace string) []string {
	nspHandler := s.getNamespace(namespace, false)
	if nspHandler != nil {
		return nspHandler.broadcast.Rooms(nil)
	}
	return nil
}

func (s *Server) ForEach(namespace string, room string, f EachFunc) bool {
	nspHandler := s.getNamespace(namespace, false)
	if nspHandler != nil {
		nspHandler.broadcast.ForEach(room, f)
		return true
	}
	return false
}

func (s *Server) serveConn(c engineio.Conn) {
	_, err := newConn(c, s.handlers)
	if err != nil {
		root := s.handlers[""]
		if root != nil && root.onError != nil {
			root.onError(nil, err)
		}
		return
	}
}

func (s *Server) getNamespace(nsp string, create bool) *namespaceHandler {
	if nsp == "/" {
		nsp = ""
	}
	ret, ok := s.handlers[nsp]
	if ok {
		return ret
	}
	if create {
		handler := newHandler()
		s.handlers[nsp] = handler
		return handler
	} else {
		return nil
	}
}
