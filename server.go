package socketio

import (
	"net/http"

	"github.com/googollee/go-socket.io/engineio"
)

// Server is a go-socket.io server.
type Server struct {
	engine *engineio.Server

	handlers map[string]*namespaceHandler
}

// NewServer returns a server.
func NewServer(c *engineio.Options) *Server {
	engine, _ := engineio.NewServer(c)

	return &Server{
		handlers: make(map[string]*namespaceHandler),
		engine:   engine,
	}
}

// Close closes server.
func (s *Server) Close() error {
	return s.engine.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.engine.ServeHTTP(w, r)
}

// OnConnect set a handler function f to handle open event for
// namespace nsp.
func (s *Server) OnConnect(namespace string, f func(Conn) error) {
	h := s.getNamespace(namespace)
	if h == nil {
		h = s.createNameSpace(namespace)
	}

	h.OnConnect(f)
}

// OnDisconnect set a handler function f to handle disconnect event for
// namespace nsp.
func (s *Server) OnDisconnect(nsp string, f func(Conn, string)) {
	h := s.getNamespace(nsp)
	if h == nil {
		h = s.createNameSpace(nsp)
	}

	h.OnDisconnect(f)
}

// OnError set a handler function f to handle error for namespace.
func (s *Server) OnError(namespace string, f func(Conn, error)) {
	h := s.getNamespace(namespace)
	if h == nil {
		h = s.createNameSpace(namespace)
	}

	h.OnError(f)
}

// OnEvent set a handler function f to handle event for namespace.
func (s *Server) OnEvent(namespace, event string, f interface{}) {
	h := s.getNamespace(namespace)
	if h == nil {
		h = s.createNameSpace(namespace)
	}

	h.OnEvent(event, f)
}

// Serve serves go-socket.io server
func (s *Server) Serve() error {
	for {
		conn, err := s.engine.Accept()
		//todo maybe need check EOF from Accept()
		if err != nil {
			return err
		}

		go s.serveConn(conn)
	}
}

// JoinRoom joins given connection to the room
func (s *Server) JoinRoom(namespace string, room string, connection Conn) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.Join(room, connection)
		return true
	}

	return false
}

// LeaveRoom leaves given connection from the room
func (s *Server) LeaveRoom(namespace string, room string, connection Conn) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.Leave(room, connection)
		return true
	}

	return false
}

// LeaveAllRooms leaves the given connection from all rooms
func (s *Server) LeaveAllRooms(namespace string, connection Conn) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.LeaveAll(connection)
		return true
	}

	return false
}

// ClearRoom clears the room
func (s *Server) ClearRoom(namespace string, room string) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.Clear(room)
		return true
	}

	return false
}

// BroadcastToRoom broadcasts given event & args to all the connections in the room
func (s *Server) BroadcastToRoom(namespace string, room, event string, args ...interface{}) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.Send(room, event, args...)
		return true
	}

	return false
}

// RoomLen gives number of connections in the room
func (s *Server) RoomLen(namespace string, room string) int {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		return nspHandler.broadcast.Len(room)
	}

	return -1
}

// Rooms gives list of all the rooms
func (s *Server) Rooms(namespace string) []string {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		return nspHandler.broadcast.Rooms(nil)
	}

	return nil
}

// Count number of connections
func (s *Server) Count() int {
	return s.engine.Count()
}

// ForEach sends data by DataFunc, if room does not exits sends nothing
func (s *Server) ForEach(namespace string, room string, f EachFunc) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.ForEach(room, f)
		return true
	}

	return false
}

func (s *Server) serveConn(conn engineio.Conn) {
	err := newConn(conn, s.handlers)

	if err != nil {
		root := s.handlers[rootNamespace]
		if root != nil && root.onError != nil {
			root.onError(nil, err)
		}
	}
}

func (s *Server) createNameSpace(nsp string) *namespaceHandler {
	if nsp == aliasRootNamespace {
		nsp = rootNamespace
	}

	handler := newNamespaceHandler()
	s.handlers[nsp] = handler

	return handler
}

func (s *Server) getNamespace(nsp string) *namespaceHandler {
	if nsp == aliasRootNamespace {
		nsp = rootNamespace
	}

	ret, ok := s.handlers[nsp]
	if !ok {
		return nil
	}

	return ret
}
