package socketio

import (
	"reflect"
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

var (
	defaultHeaderType = []reflect.Type{reflect.TypeOf("")}
)
