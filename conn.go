package socketio

import (
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/googollee/go-socket.io/parser"

	"gopkg.in/googollee/go-engine.io.v1"
)

type Conn interface {
	ID() string
	Close() error
	URL() url.URL
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	RemoteHeader() http.Header

	SetContext(v interface{})
	Context() interface{}
	Namespace() string
	Emit(msg string, v ...interface{})
}

type errorMessage struct {
	namespace string
	error
}

type writePacket struct {
	header parser.Header
	data   []interface{}
}

type conn struct {
	engineio.Conn
	encoder    *parser.Encoder
	decoder    *parser.Decoder
	errorChan  chan errorMessage
	writeChan  chan writePacket
	quitChan   chan struct{}
	handlers   map[string]*namespaceHandler
	namespaces map[string]*namespaceConn
	closeOnce  sync.Once
	id         uint64
}

func newConn(c engineio.Conn, handlers map[string]*namespaceHandler) (*conn, error) {
	ret := &conn{
		Conn:       c,
		encoder:    parser.NewEncoder(c),
		decoder:    parser.NewDecoder(c),
		errorChan:  make(chan errorMessage),
		writeChan:  make(chan writePacket),
		quitChan:   make(chan struct{}),
		handlers:   handlers,
		namespaces: make(map[string]*namespaceConn),
	}
	if err := ret.connect(); err != nil {
		ret.Close()
		return nil, err
	}
	go ret.serveError()
	go ret.serveWrite()
	go ret.serveRead()
	return ret, nil
}

func (c *conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		err = c.Conn.Close()
		close(c.quitChan)
	})
	return err
}

func (c *conn) connect() error {
	header := parser.Header{
		Type: parser.Connect,
	}
	if err := c.encoder.Encode(header, nil); err != nil {
		return err
	}

	c.namespaces[""] = newNamespaceConn(c, "/")

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
	onErr := errorMessage{
		namespace: namespace,
		error:     err,
	}
	select {
	case c.errorChan <- onErr:
	case <-c.quitChan:
		return
	}
}

func (c *conn) parseArgs(types []reflect.Type) ([]reflect.Value, error) {
	return c.decoder.DecodeArgs(types)
}

func (c *conn) serveError() {
	defer c.Close()
	for {
		select {
		case <-c.quitChan:
			return
		case msg := <-c.errorChan:
			if handler := c.namespace(msg.namespace); handler != nil {
				handler.onError(msg.error)
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

func (c *conn) serveRead() {
	defer c.Close()
	var header parser.Header
	var event string
	for {
		if err := c.decoder.DecodeHeader(&header, &event); err != nil {
			c.onError("", err)
			return
		}
		if header.Namespace == "/" {
			header.Namespace = ""
		}
		switch header.Type {
		case parser.Ack:
			conn, ok := c.namespaces[header.Namespace]
			if !ok {
				c.decoder.DiscardLast()
				continue
			}
			conn.dispatch(header)
		case parser.Event:
			conn, ok := c.namespaces[header.Namespace]
			if !ok {
				c.decoder.DiscardLast()
				continue
			}
			handler, ok := c.handlers[header.Namespace]
			if !ok {
				c.decoder.DiscardLast()
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
			if _, ok := c.namespaces[header.Namespace]; !ok {
				c.namespaces[header.Namespace] = newNamespaceConn(c, header.Namespace)
			}
			c.write(header, nil)
		case parser.Disconnect:
			types := []reflect.Type{reflect.TypeOf("")}
			args, err := c.decoder.DecodeArgs(types)
			if err != nil {
				c.onError(header.Namespace, err)
				return
			}
			conn, ok := c.namespaces[header.Namespace]
			if !ok {
				c.decoder.DiscardLast()
				continue
			}
			delete(c.namespaces, header.Namespace)
			handler, ok := c.handlers[header.Namespace]
			if ok {
				handler.dispatch(conn, header, "", args)
			}
		}
	}
}

func (c *conn) namespace(nsp string) *namespaceHandler {
	return c.handlers[nsp]
}
