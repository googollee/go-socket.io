package socketio

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/armon/go-radix"
	"github.com/googollee/go-socket.io/parser"
)

type namespaceHandler struct {
	broadcast Broadcast

	events         map[string]*funcHandler
	wildcardEvents *radix.Tree
	eventsLock     sync.RWMutex

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
		broadcast:      broadcast,
		events:         make(map[string]*funcHandler),
		wildcardEvents: radix.New(),
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
	ef := newEventFunc(f)
	nh.events[event] = ef

	if strings.HasSuffix(event, "*") {
		nh.wildcardEvents.Insert(strings.TrimSuffix(event, "*"), ef)
	}
}

func (nh *namespaceHandler) GetEventHandler(event string) *funcHandler {
	nh.eventsLock.Lock()
	defer nh.eventsLock.Unlock()
	eventHandler, ok := nh.events[event]

	if !ok {
		// If not found one directly check for longest wildcard match
		_, wcEH, ok := nh.wildcardEvents.LongestPrefix(event)
		if !ok {
			return nil
		}
		eventHandler = wcEH.(*funcHandler)
	}
	return eventHandler
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

func getDispatchMessage(args ...reflect.Value) string {
	var msg string
	if len(args) > 0 {
		msg = args[0].Interface().(string)
	}

	return msg
}
