package socketio

import (
	"errors"
	"reflect"
	"sync"

	"github.com/googollee/go-socket.io/parser"
)

type namespaceHandler struct {
	broadcast Broadcast

	events     map[string]*funcHandler
	eventsLock sync.RWMutex

	onConnect    func(conn Conn) error
	onDisconnect func(conn Conn, msg string)
	onError      func(conn Conn, err error)
}

func newNamespaceHandler(nsp string, adapterOpts *RedisAdapterOptions) *namespaceHandler {
	var broadcast Broadcast
	if adapterOpts == nil {
		broadcast = newBroadcast()
	} else {
		broadcast, _ = newRedisBroadcast(nsp, adapterOpts)
	}

	return &namespaceHandler{
		broadcast: broadcast,
		events:    make(map[string]*funcHandler),
	}
}

func (nh *namespaceHandler) OnConnect(f func(Conn) error) {
	nh.onConnect = f
}

func (nh *namespaceHandler) OnDisconnect(f func(Conn, string)) {
	nh.onDisconnect = f
}

func (nh *namespaceHandler) OnError(f func(Conn, error)) {
	nh.onError = f
}

func (nh *namespaceHandler) OnEvent(event string, f interface{}) {
	nh.eventsLock.Lock()
	defer nh.eventsLock.Unlock()

	nh.events[event] = newEventFunc(f)
}

func (nh *namespaceHandler) getEventTypes(event string) []reflect.Type {
	nh.eventsLock.RLock()
	namespaceHandler := nh.events[event]
	nh.eventsLock.RUnlock()

	if namespaceHandler != nil {
		return namespaceHandler.argTypes
	}

	return nil
}

func (nh *namespaceHandler) dispatch(conn Conn, header parser.Header, args ...reflect.Value) ([]reflect.Value, error) {
	switch header.Type {
	case parser.Connect:
		if nh.onConnect != nil {
			return nil, nh.onConnect(conn)
		}
		return nil, nil

	case parser.Disconnect:
		if nh.onDisconnect != nil {
			nh.onDisconnect(conn, getDispatchMessage(args...))
		}
		return nil, nil

	case parser.Error:
		if nh.onError != nil {
			msg := getDispatchMessage(args...)
			if msg == "" {
				msg = "parser error dispatch"
			}
			nh.onError(conn, errors.New(msg))
		}
	}

	return nil, parser.ErrInvalidPacketType
}

func (nh *namespaceHandler) dispatchEvent(conn Conn, event string, args ...reflect.Value) ([]reflect.Value, error) {
	nh.eventsLock.RLock()
	namespaceHandler := nh.events[event]
	nh.eventsLock.RUnlock()

	if namespaceHandler == nil {
		return nil, nil
	}

	return namespaceHandler.Call(append([]reflect.Value{reflect.ValueOf(conn)}, args...))
}

func getDispatchMessage(args ...reflect.Value) string {
	var msg string
	if len(args) > 0 {
		msg = args[0].Interface().(string)
	}

	return msg
}
