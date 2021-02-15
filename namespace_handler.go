package socketio

import (
	"errors"
	"reflect"
	"sync"

	"github.com/googollee/go-socket.io/parser"
)

type namespaceHandler struct {
	broadcast Broadcast

	eventsMu sync.RWMutex
	events map[string]*funcHandler

	onConnect    func(c Conn) error
	onDisconnect func(c Conn, msg string)
	onError      func(c Conn, err error)
}

func newNamespaceHandler() *namespaceHandler {
	return &namespaceHandler{
		broadcast: newBroadcast(),
		events:    make(map[string]*funcHandler),
	}
}

func (h *namespaceHandler) OnConnect(f func(Conn) error) {
	h.onConnect = f
}

func (h *namespaceHandler) OnDisconnect(f func(Conn, string)) {
	h.onDisconnect = f
}

func (h *namespaceHandler) OnError(f func(Conn, error)) {
	h.onError = f
}

func (h *namespaceHandler) OnEvent(event string, f interface{}) {
	h.eventsMu.Lock()
	defer h.eventsMu.Unlock()
	h.events[event] = newEventFunc(f)
}

func (h *namespaceHandler) getTypes(header parser.Header, event string) []reflect.Type {
	switch header.Type {
	case parser.Error:
		fallthrough
	case parser.Disconnect:
		return []reflect.Type{reflect.TypeOf("")}
	case parser.Event:
		h.eventsMu.RLock()
		namespaceHandler := h.events[event]
		h.eventsMu.RUnlock()
		if namespaceHandler == nil {
			return nil
		}
		return namespaceHandler.argTypes
	}

	return nil
}

//todo maybe refactor this
func (h *namespaceHandler) dispatch(c Conn, header parser.Header, event string, args []reflect.Value) ([]reflect.Value, error) {
	switch header.Type {
	case parser.Connect:
		if h.onConnect != nil {
			return nil, h.onConnect(c)
		}
		return nil, nil
	case parser.Disconnect:
		var msg string

		if len(args) > 0 {
			msg = args[0].Interface().(string)
		}
		if h.onDisconnect != nil {
			h.onDisconnect(c, msg)
		}

		return nil, nil
	case parser.Error:
		var msg string

		if len(args) > 0 {
			msg = args[0].Interface().(string)
		}

		if h.onError != nil {
			h.onError(c, errors.New(msg))
		}
	case parser.Event:
		h.eventsMu.RLock()
		namespaceHandler := h.events[event]
		h.eventsMu.RUnlock()
		if namespaceHandler == nil {
			return nil, nil
		}

		return namespaceHandler.Call(append([]reflect.Value{reflect.ValueOf(c)}, args...))
	}

	return nil, parser.ErrInvalidPacketType
}
