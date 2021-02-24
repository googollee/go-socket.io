package socketio

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/parser"
)

// Conn is a connection in go-socket.io
type Conn interface {
	io.Closer
	Namespace

	// ID returns session id
	ID() string
	URL() url.URL
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	RemoteHeader() http.Header
}

type writePacket struct {
	header parser.Header

	data []interface{}
}

type namespaces struct {
	mu sync.RWMutex
	namespaces map[string]*namespaceConn
}

func newNamespaces() *namespaces {
	return &namespaces{
		namespaces: make(map[string]*namespaceConn),
	}
}

func (n *namespaces) Get(ns string) (*namespaceConn, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	namespace, ok := n.namespaces[ns]
	return namespace, ok
}

func (n *namespaces) Set(ns string, conn *namespaceConn) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.namespaces[ns] = conn
}

func (n *namespaces) Delete(ns string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.namespaces, ns)
}

func (n *namespaces) Range(fn func(ns string, nc *namespaceConn)) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for ns, nc := range n.namespaces {
		fn(ns, nc)
	}
}

type conn struct {
	engineio.Conn

	id         uint64
	handlers   *namespaceHandlers
	namespaces *namespaces

	encoder *parser.Encoder
	decoder *parser.Decoder

	errorChan chan error
	writeChan chan writePacket
	quitChan  chan struct{}

	closeOnce sync.Once
}

func newConn(c engineio.Conn, handlers *namespaceHandlers) error {
	ret := &conn{
		Conn:       c,
		encoder:    parser.NewEncoder(c),
		decoder:    parser.NewDecoder(c),
		errorChan:  make(chan error),
		writeChan:  make(chan writePacket),
		quitChan:   make(chan struct{}),
		handlers:   handlers,
		namespaces: newNamespaces(),
	}

	if err := ret.connect(); err != nil {
		_ = ret.Close()
		return err
	}

	return nil
}

func (c *conn) Close() error {
	var err error

	c.closeOnce.Do(func() {
		//for each namespace, leave all rooms, and call the disconnect handler.
		c.namespaces.Range(func(ns string, nc *namespaceConn) {
			nc.LeaveAll()

			if nh, _ := c.handlers.Get(ns); nh != nil && nh.onDisconnect != nil {
				nh.onDisconnect(nc, clientDisconnectMsg)
			}
		})
		err = c.Conn.Close()

		close(c.quitChan)
	})

	return err
}

func (c *conn) connect() error {
	rootHandler, ok := c.handlers.Get(rootNamespace)
	if !ok {
		return errUnavailableRootHandler
	}

	root := newNamespaceConn(c, aliasRootNamespace, rootHandler.broadcast)
	c.namespaces.Set(rootNamespace, root)

	root.Join(root.ID())
	
	c.namespaces.Range(func(ns string, nc *namespaceConn) {
		nc.SetContext(c.Conn.Context())
	})

	header := parser.Header{
		Type: parser.Connect,
	}

	if err := c.encoder.Encode(header, nil); err != nil {
		return err
	}
	handler, ok := c.handlers.Get(header.Namespace)

	go c.serveError()
	go c.serveWrite()
	go c.serveRead()

	if ok {
		_, err := handler.dispatch(root, header, "", nil)
		return err
	}

	return nil
}

func (c *conn) nextID() uint64 {
	c.id++

	return c.id
}

func (c *conn) write(header parser.Header, args []reflect.Value) {
	data := make([]interface{}, len(args))

	for i := range data {
		data[i] = args[i].Interface()
	}
	pkg := writePacket{
		header: header,
		data:   data,
	}

	select {
	case c.writeChan <- pkg:
	case <-c.quitChan:
		return
	}
}

func (c *conn) onError(namespace string, err error) {
	select {
	case c.errorChan <- newErrorMessage(namespace, err):
	case <-c.quitChan:
		return
	}
}

func (c *conn) serveError() {
	defer c.Close()

	for {
		select {
		case <-c.quitChan:
			return
		case err := <-c.errorChan:
			errMsg, ok := err.(errorMessage)
			// todo add log
			if !ok {
				continue
			}
			if handler := c.namespace(errMsg.namespace); handler != nil {
				if handler.onError != nil {
					ns, ok := c.namespaces.Get(errMsg.namespace)
					if !ok {
						continue
					}
					handler.onError(ns, errMsg.err)
				}
			}
		}
	}
}

func (c *conn) serveWrite() {
	defer c.Close()

	for {
		select {
		case <-c.quitChan:
			return
		case pkg := <-c.writeChan:
			if err := c.encoder.Encode(pkg.header, pkg.data); err != nil {
				c.onError(pkg.header.Namespace, err)
			}
		}
	}
}

//todo maybe refactor this
func (c *conn) serveRead() {
	defer c.Close()

	var event string

	for {
		var header parser.Header

		if err := c.decoder.DecodeHeader(&header, &event); err != nil {
			c.onError(rootNamespace, err)
			return
		}

		if header.Namespace == aliasRootNamespace {
			header.Namespace = rootNamespace
		}

		switch header.Type {
		case parser.Ack:
			conn, ok := c.namespaces.Get(header.Namespace)
			if !ok {
				_ = c.decoder.DiscardLast()
				continue
			}
			conn.dispatch(header)
		case parser.Event:
			conn, ok := c.namespaces.Get(header.Namespace)
			if !ok {
				_ = c.decoder.DiscardLast()
				continue
			}
			handler, ok := c.handlers.Get(header.Namespace)
			if !ok {
				_ = c.decoder.DiscardLast()
				continue
			}
			types := handler.getTypes(header, event)
			args, err := c.decoder.DecodeArgs(types)
			if err != nil {
				c.onError(header.Namespace, err)
				return
			}
			ret, err := handler.dispatch(conn, header, event, args)
			if err != nil {
				c.onError(header.Namespace, err)
				return
			}
			if len(ret) > 0 {
				header.Type = parser.Ack
				c.write(header, ret)
			}
		case parser.Connect:
			if err := c.decoder.DiscardLast(); err != nil {
				c.onError(header.Namespace, err)
				return
			}

			handler, ok := c.handlers.Get(header.Namespace)
			if ok {
				conn, ok := c.namespaces.Get(header.Namespace)
				if !ok {
					conn = newNamespaceConn(c, header.Namespace, handler.broadcast)
					c.namespaces.Set(header.Namespace, conn)
					conn.Join(c.ID())
				}
				_, _ = handler.dispatch(conn, header, "", nil)

				//todo leave default room?!
			} else {
				c.onError(header.Namespace, errFailedConnetNamespace)
				return
			}
			c.write(header, nil)
		case parser.Disconnect:
			types := []reflect.Type{reflect.TypeOf("")}
			args, err := c.decoder.DecodeArgs(types)
			if err != nil {
				c.onError(header.Namespace, err)
				return
			}
			conn, ok := c.namespaces.Get(header.Namespace)
			if !ok {
				_ = c.decoder.DiscardLast()
				continue
			}

			conn.LeaveAll()
			c.namespaces.Delete(header.Namespace)
			handler, ok := c.handlers.Get(header.Namespace)
			if ok {
				_, _ = handler.dispatch(conn, header, "", args)
			}
		}
	}
}

func (c *conn) namespace(nsp string) *namespaceHandler {
	handler, _ := c.handlers.Get(nsp)
	return handler
}
