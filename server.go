package socketio

import (
	"errors"
	"net/http"

	"github.com/gomodule/redigo/redis"

	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/logger"
	"github.com/googollee/go-socket.io/parser"
)

// Server is a go-socket.io server.
type Server struct {
	engine *engineio.Server

	handlers *namespaceHandlers

	redisAdapter *RedisAdapterOptions
}

// NewServer returns a server.
func NewServer(opts *engineio.Options) *Server {
	return &Server{
		handlers: newNamespaceHandlers(),
		engine:   engineio.NewServer(opts),
	}
}

// Adapter sets redis broadcast adapter.
func (s *Server) Adapter(opts *RedisAdapterOptions) (bool, error) {
	opts = getOptions(opts)
	var redisOpts []redis.DialOption
	if len(opts.Password) > 0 {
		redisOpts = append(redisOpts, redis.DialPassword(opts.Password))
	}
	if opts.DB > 0 {
		redisOpts = append(redisOpts, redis.DialDatabase(opts.DB))
	}

	conn, err := redis.Dial(opts.Network, opts.getAddr(), redisOpts...)
	if err != nil {
		return false, err
	}

	s.redisAdapter = opts

	return true, conn.Close()
}

// Close closes server.
func (s *Server) Close() error {
	return s.engine.Close()
}

// ServeHTTP dispatches the request to the handler whose pattern most closely matches the request URL.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.engine.ServeHTTP(w, r)
}

// OnConnect set a handler function f to handle open event for namespace.
func (s *Server) OnConnect(namespace string, f func(Conn) error) {
	h := s.getNamespace(namespace)
	if h == nil {
		h = s.createNamespace(namespace)
	}

	h.OnConnect(f)
}

// OnDisconnect set a handler function f to handle disconnect event for namespace.
func (s *Server) OnDisconnect(namespace string, f func(Conn, string)) {
	h := s.getNamespace(namespace)
	if h == nil {
		h = s.createNamespace(namespace)
	}

	h.OnDisconnect(f)
}

// OnError set a handler function f to handle error for namespace.
func (s *Server) OnError(namespace string, f func(Conn, error)) {
	h := s.getNamespace(namespace)
	if h == nil {
		h = s.createNamespace(namespace)
	}

	h.OnError(f)
}

// OnEvent set a handler function f to handle event for namespace.
func (s *Server) OnEvent(namespace, event string, f interface{}) {
	h := s.getNamespace(namespace)
	if h == nil {
		h = s.createNamespace(namespace)
	}

	h.OnEvent(event, f)
}

// Serve serves go-socket.io server.
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

// JoinRoom joins given connection to the room.
func (s *Server) JoinRoom(namespace string, room string, connection Conn) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.Join(room, connection)
		return true
	}

	return false
}

// LeaveRoom leaves given connection from the room.
func (s *Server) LeaveRoom(namespace string, room string, connection Conn) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.Leave(room, connection)
		return true
	}

	return false
}

// LeaveAllRooms leaves the given connection from all rooms.
func (s *Server) LeaveAllRooms(namespace string, connection Conn) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.LeaveAll(connection)
		return true
	}

	return false
}

// ClearRoom clears the room.
func (s *Server) ClearRoom(namespace string, room string) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.Clear(room)
		return true
	}

	return false
}

// BroadcastToRoom broadcasts given event & args to all the connections in the room.
func (s *Server) BroadcastToRoom(namespace string, room, event string, args ...interface{}) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.Send(room, event, args...)
		return true
	}

	return false
}

// BroadcastToNamespace broadcasts given event & args to all the connections in the same namespace.
func (s *Server) BroadcastToNamespace(namespace string, event string, args ...interface{}) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.SendAll(event, args...)
		return true
	}

	return false
}

// RoomLen gives number of connections in the room.
func (s *Server) RoomLen(namespace string, room string) int {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		return nspHandler.broadcast.Len(room)
	}

	return -1
}

// Rooms gives list of all the rooms.
func (s *Server) Rooms(namespace string) []string {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		return nspHandler.broadcast.Rooms(nil)
	}

	return nil
}

// Count number of connections.
func (s *Server) Count() int {
	return s.engine.Count()
}

// Remove session from sessions pool. Fixed the sessions map leak(connections, mem).
func (s *Server) Remove(sid string) {
	s.engine.Remove(sid)
}

// ForEach sends data by DataFunc, if room does not exit sends anything.
func (s *Server) ForEach(namespace string, room string, f EachFunc) bool {
	nspHandler := s.getNamespace(namespace)
	if nspHandler != nil {
		nspHandler.broadcast.ForEach(room, f)
		return true
	}

	return false
}

func (s *Server) serveConn(conn engineio.Conn) {
	c := newConn(conn, s.handlers)
	if err := c.connect(); err != nil {
		_ = c.Close()
		if root, ok := s.handlers.Get(rootNamespace); ok && root.onError != nil {
			root.onError(nil, err)
		}

		return
	}

	go s.serveError(c)
	go s.serveWrite(c)
	go s.serveRead(c)
}

func (s *Server) serveError(c *conn) {
	defer func() {
		if err := c.Close(); err != nil {
			logger.Error("close connect:", err)
		}

		s.engine.Remove(c.Conn.ID())
	}()

	for {
		select {
		case <-c.quitChan:
			return
		case err := <-c.errorChan:
			var errMsg *errorMessage
			if !errors.As(err, &errMsg) {
				continue
			}

			if handler := c.namespace(errMsg.namespace); handler != nil {
				if handler.onError != nil {
					nsConn, ok := c.namespaces.Get(errMsg.namespace)
					if !ok {
						continue
					}
					handler.onError(nsConn, errMsg.err)
				}
			}
		}
	}
}

func (s *Server) serveWrite(c *conn) {
	defer func() {
		if err := c.Close(); err != nil {
			logger.Error("close connect:", err)
		}

		s.engine.Remove(c.Conn.ID())
	}()

	for {
		select {
		case <-c.quitChan:
			return
		case pkg := <-c.writeChan:
			if err := c.encoder.Encode(pkg.Header, pkg.Data); err != nil {
				c.onError(pkg.Header.Namespace, err)
			}
		}
	}
}

func (s *Server) serveRead(c *conn) {
	defer func() {
		if err := c.Close(); err != nil {
			logger.Error("close connect:", err)
		}

		s.engine.Remove(c.Conn.ID())
	}()

	var event string

	for {
		var header parser.Header

		if err := c.decoder.DecodeHeader(&header, &event); err != nil {
			logger.Error("DecodeHeader Error in serveRead", err)
			c.onError(rootNamespace, err)
			return
		}

		if header.Namespace == aliasRootNamespace {
			header.Namespace = rootNamespace
		}

		var err error
		switch header.Type {
		case parser.Ack, parser.Connect, parser.Disconnect:
			handler, ok := readHandlerMapping[header.Type]
			if !ok {
				return
			}

			err = handler(c, header)
		case parser.Event:
			err = eventPacketHandler(c, event, header)
		}

		if err != nil {
			logger.Error("serve read:", err)

			return
		}
	}
}

func (s *Server) createNamespace(nsp string) *namespaceHandler {
	if nsp == aliasRootNamespace {
		nsp = rootNamespace
	}

	handler := newNamespaceHandler(nsp, s.redisAdapter)
	s.handlers.Set(nsp, handler)

	return handler
}

func (s *Server) getNamespace(nsp string) *namespaceHandler {
	if nsp == aliasRootNamespace {
		nsp = rootNamespace
	}

	ret, ok := s.handlers.Get(nsp)
	if !ok {
		return nil
	}

	return ret
}
