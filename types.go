package socketio

import (
	"reflect"

	"github.com/vchitai/go-socket.io/v4/parser"
)

// namespace
const (
	aliasRootNamespace = "/"
	rootNamespace      = ""
)

// message
const (
	clientDisconnectMsg = "client namespace disconnect"
)

type readHandler func(c *conn, header parser.Header) error

var (
	defaultHeaderType = []reflect.Type{reflect.TypeOf(make(map[string]interface{}))}
)
