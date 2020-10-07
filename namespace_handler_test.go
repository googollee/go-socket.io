package socketio

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/googollee/go-socket.io/parser"
)

func TestNamespaceHandler(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	h := newNamespaceHandler()

	onConnectCalled := false
	h.OnConnect(func(c Conn) error {
		onConnectCalled = true
		return nil
	})

	disconnectMsg := ""
	h.OnDisconnect(func(c Conn, reason string) {
		disconnectMsg = reason
	})

	var onerror error
	h.OnError(func(conn Conn, err error) {
		onerror = err
	})

	header := parser.Header{}

	header.Type = parser.Connect
	args := h.getTypes(header, "")

	should.Nil(args)

	_, err := h.dispatch(&namespaceConn{}, header, "", nil)
	must.NoError(err)

	should.True(onConnectCalled)

	header.Type = parser.Disconnect
	args = h.getTypes(header, "")

	should.Equal([]reflect.Type{reflect.TypeOf("")}, args)

	_, err = h.dispatch(&namespaceConn{}, header, "", []reflect.Value{reflect.ValueOf("disconn")})
	must.NoError(err)

	should.Equal("disconn", disconnectMsg)

	header.Type = parser.Error
	args = h.getTypes(header, "")

	should.Equal([]reflect.Type{reflect.TypeOf("")}, args)

	_, err = h.dispatch(&namespaceConn{}, header, "", []reflect.Value{reflect.ValueOf("failed")})
	must.Error(err)

	should.Equal(onerror.Error(), "failed")

	header.Type = parser.Event
	args = h.getTypes(header, "nonexist")

	should.Nil(args)

	ret, err := h.dispatch(&namespaceConn{}, header, "nonexist", nil)

	must.NoError(err)
	should.Nil(ret)
}

func TestNamespaceHandlerEvent(t *testing.T) {
	tests := []struct {
		name string

		events   []string
		handlers []interface{}

		event string
		args  []interface{}

		ok  bool
		ret []interface{}
	}{
		{
			name: "string handler",

			events: []string{"e", "n"},
			handlers: []interface{}{
				func(c Conn, str string) string {
					return "handled " + str
				},
				func(c Conn) {},
			},

			event: "e",
			args:  []interface{}{"str"},

			ok:  true,
			ret: []interface{}{"handled str"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			should := assert.New(t)
			must := require.New(t)

			h := newNamespaceHandler()
			for i, e := range test.events {
				h.OnEvent(e, test.handlers[i])
			}

			header := parser.Header{
				Type: parser.Event,
			}
			target := make([]reflect.Type, len(test.args))
			args := make([]reflect.Value, len(test.args))

			for i := range test.args {
				target[i] = reflect.TypeOf(test.args[i])
				args[i] = reflect.ValueOf(test.args[i])
			}

			types := h.getTypes(header, test.event)
			should.Equal(target, types)

			ret, err := h.dispatch(&namespaceConn{}, header, test.event, args)
			must.NoError(err)

			res := make([]interface{}, len(ret))
			for i := range ret {
				res[i] = ret[i].Interface()
			}

			should.Equal(test.ret, res)
		})
	}
}
