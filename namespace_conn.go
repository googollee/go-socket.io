package socketio

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/googollee/go-socket.io/parser"
)

// Namespace
type Namespace interface {
	// Context of this connection. You can save one context for one
	// connection, and share it between all handlers. The handlers
	// is called in one goroutine, so no need to lock context if it
	// only be accessed in one connection.
	SetContext(ctx interface{})
	Context() interface{}
	Namespace() string
	Emit(eventName string, v ...interface{})

	// Broadcast server side apis
	Join(room string)
	Leave(room string)
	LeaveAll()
	Rooms() []string
}

type namespaceConn struct {
	*conn
	broadcast Broadcast

	namespace string
	context   interface{}

	ack sync.Map
}

func newNamespaceConn(conn *conn, namespace string, broadcast Broadcast) *namespaceConn {
	return &namespaceConn{
		conn:      conn,
		namespace: namespace,
		broadcast: broadcast,
	}
}

func (c *namespaceConn) SetContext(ctx interface{}) {
	c.context = ctx
}

func (c *namespaceConn) Context() interface{} {
	return c.context
}

func (c *namespaceConn) Namespace() string {
	return c.namespace
}

func (c *namespaceConn) Emit(eventName string, v ...interface{}) {
	header := parser.Header{
		Type: parser.Event,
	}
	if c.namespace != aliasRootNamespace {
		header.Namespace = c.namespace
	}

	if l := len(v); l > 0 {
		last := v[l-1]
		lastV := reflect.TypeOf(last)

		if lastV.Kind() == reflect.Func {
			f := newAckFunc(last)

			header.ID = c.conn.nextID()
			header.NeedAck = true

			c.ack.Store(header.ID, f)
			v = v[:l-1]
		}
	}

	args := make([]reflect.Value, len(v)+1)
	args[0] = reflect.ValueOf(eventName)

	for i := 1; i < len(args); i++ {
		args[i] = reflect.ValueOf(v[i-1])
	}

	c.conn.write(header, args)
}

func (c *namespaceConn) Join(room string) {
	c.broadcast.Join(room, c)
}

func (c *namespaceConn) Leave(room string) {
	c.broadcast.Leave(room, c)
}

func (c *namespaceConn) LeaveAll() {
	c.broadcast.LeaveAll(c)
}

func (c *namespaceConn) Rooms() []string {
	return c.broadcast.Rooms(c)
}

func (c *namespaceConn) dispatch(header parser.Header) {
	if header.Type != parser.Ack {
		return
	}

	rawFunc, ok := c.ack.Load(header.ID)
	if ok {
		f, ok := rawFunc.(*funcHandler)
		if !ok {
			c.conn.onError(c.namespace, fmt.Errorf("incorrect data stored for header %d", header.ID))
			return
		}

		c.ack.Delete(header.ID)

		args, err := c.conn.parseArgs(f.argTypes)
		if err != nil {
			c.conn.onError(c.namespace, err)
			return
		}
		if _, err := f.Call(args); err != nil {
			c.conn.onError(c.namespace, err)
			return
		}
	}
	return
}

func (c *conn) parseArgs(types []reflect.Type) ([]reflect.Value, error) {
	return c.decoder.DecodeArgs(types)
}
