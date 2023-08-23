package socketio

import (
	"fmt"
	"reflect"
)

type funcHandler struct {
	argTypes        []reflect.Type
	f               reflect.Value
	hasConn         bool
	hasEventRequest bool
}

type EventRequest interface {
	Event() string
}

type eventRequest struct {
	event string
}

func (e *eventRequest) Event() string {
	return e.event
}

func (h *funcHandler) CallAck(args []reflect.Value) (ret []reflect.Value, err error) {

	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("event call error: %s, args %v", r, args)
			}
		}
	}()

	ret = h.f.Call(args)
	return
}

func (h *funcHandler) CallEvent(c Conn, event string, args []reflect.Value) (ret []reflect.Value, err error) {

	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("event call error: %s, args %v", r, args)
			}
		}
	}()

	if h.hasEventRequest {
		args = append([]reflect.Value{reflect.ValueOf(&eventRequest{event})}, args...)
	}

	if h.hasConn {
		args = append([]reflect.Value{reflect.ValueOf(c)}, args...)
	}

	ret = h.f.Call(args)
	return
}

func newEventFunc(f interface{}) *funcHandler {
	fv := reflect.ValueOf(f)

	if fv.Kind() != reflect.Func {
		panic("event handler must be a func.")
	}
	ft := fv.Type()

	// Function type can be
	// func(...)
	// func(socketio.Conn, ...)
	// func(socketio.Conn, EventRequest ...)

	hasConn := false
	hasEventRequest := false

	switch ft.NumIn() {
	case 0:
		hasConn = false
		hasEventRequest = false
	case 1:
		hasConn = implementsConn(ft.In(0))
		hasEventRequest = false
	default:
		hasConn = implementsConn(ft.In(0))
		hasEventRequest = implementsEventRequest(ft.In(1))
	}

	// Finding the number of remaining arguments
	argsStart := 0
	if hasConn {
		argsStart += 1
	}
	if hasEventRequest {
		argsStart += 1
	}

	argTypes := make([]reflect.Type, ft.NumIn()-argsStart)
	for i := range argTypes {
		argTypes[i] = ft.In(i + argsStart)
	}

	if len(argTypes) == 0 {
		argTypes = nil
	}

	return &funcHandler{
		argTypes:        argTypes,
		f:               fv,
		hasConn:         hasConn,
		hasEventRequest: hasEventRequest,
	}
}

func implementsConn(argumentType reflect.Type) bool {
	connType := reflect.TypeOf((*Conn)(nil)).Elem()

	if argumentType.Kind() != reflect.Interface || !connType.Implements(argumentType) || !argumentType.Implements(connType) {
		return false
	}

	return true
}

func implementsEventRequest(argumentType reflect.Type) bool {
	eventRequestType := reflect.TypeOf((*EventRequest)(nil)).Elem()

	if argumentType.Kind() != reflect.Interface || !eventRequestType.Implements(argumentType) || !argumentType.Implements(eventRequestType) {
		return false
	}

	return true
}

func newAckFunc(f interface{}) *funcHandler {
	fv := reflect.ValueOf(f)

	if fv.Kind() != reflect.Func {
		panic("ack callback must be a func.")
	}

	ft := fv.Type()
	argTypes := make([]reflect.Type, ft.NumIn())

	for i := range argTypes {
		argTypes[i] = ft.In(i)
	}
	if len(argTypes) == 0 {
		argTypes = nil
	}

	return &funcHandler{
		argTypes: argTypes,
		f:        fv,
	}
}
