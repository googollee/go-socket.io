package socketio

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/googollee/go-socket.io/parser"
)

// Namespace describes a communication channel that allows you to split the logic of your application
// over a single shared connection.
type Namespace interface {
	// Context of this connection. You can save one context for one
	// connection, and share it between all handlers. The handlers
	// is called in one goroutine, so no need to lock context if it
	// only be accessed in one connection.
	Context() interface{}
	SetContext(ctx interface{})

	Namespace() string
	Emit(eventName string, v ...interface{})

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

func (nc *namespaceConn) SetContext(ctx interface{}) {
	nc.context = ctx
}

func (nc *namespaceConn) Context() interface{} {
	return nc.context
}

func (nc *namespaceConn) Namespace() string {
	return nc.namespace
}

func (nc *namespaceConn) Emit(eventName string, v ...interface{}) {
	header := parser.Header{
		Type: parser.Event,
	}

	if nc.namespace != aliasRootNamespace {
		header.Namespace = nc.namespace
	}

	if l := len(v); l > 0 {
		last := v[l-1]
		lastV := reflect.TypeOf(last)

		if lastV.Kind() == reflect.Func {
			f := newAckFunc(last)

			header.ID = nc.conn.nextID()
			header.NeedAck = true

			nc.ack.Store(header.ID, f)
			v = v[:l-1]
		}
	}

	args := make([]reflect.Value, len(v)+1)
	args[0] = reflect.ValueOf(eventName)

	for i := 1; i < len(args); i++ {
		args[i] = reflect.ValueOf(v[i-1])
	}

	nc.conn.write(header, args...)
}

func (nc *namespaceConn) Join(room string) {
	nc.broadcast.Join(room, nc)
}

func (nc *namespaceConn) Leave(room string) {
	nc.broadcast.Leave(room, nc)
}

func (nc *namespaceConn) LeaveAll() {
	nc.broadcast.LeaveAll(nc)
}

func (nc *namespaceConn) Rooms() []string {
	return nc.broadcast.Rooms(nc)
}

func (nc *namespaceConn) dispatch(header parser.Header) {
	if header.Type != parser.Ack {
		return
	}

	rawFunc, ok := nc.ack.Load(header.ID)
	if ok {
		f, ok := rawFunc.(*funcHandler)
		if !ok {
			nc.conn.onError(nc.namespace, fmt.Errorf("incorrect data stored for header %d", header.ID))
			return
		}

		nc.ack.Delete(header.ID)

		args, err := nc.conn.parseArgs(f.argTypes)
		if err != nil {
			nc.conn.onError(nc.namespace, err)
			return
		}
		if _, err := f.Call(args); err != nil {
			nc.conn.onError(nc.namespace, err)
			return
		}
	}
	return
}
