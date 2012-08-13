package socketio

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
)

type Event struct {
	Name string            `json:"name"`
	Args []json.RawMessage `json:"args"`
}

type eventHandler struct {
	fn   reflect.Value
	args []reflect.Type
}

type EventEmitter struct {
	mutex      sync.Mutex
	events     map[string][]*eventHandler
	eventsOnce map[string][]*eventHandler
}

func NewEventEmitter() *EventEmitter {
	return &EventEmitter{events: make(map[string][]*eventHandler), eventsOnce: make(map[string][]*eventHandler)}
}

// global cache
var eventHandlerCache = &struct {
	sync.RWMutex
	cache map[uintptr]*eventHandler
}{cache: make(map[uintptr]*eventHandler)}

func genEventHandler(fn interface{}) (handler *eventHandler, err error) {
	// if a handler have been generated before, use it first
	fnValue := reflect.ValueOf(fn)
	eventHandlerCache.RLock()
	if handler, ok := eventHandlerCache.cache[fnValue.Pointer()]; ok {
		eventHandlerCache.RUnlock()
		return handler, nil
	}
	eventHandlerCache.RUnlock()
	handler = new(eventHandler)
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		err = fmt.Errorf("%v is not a function", fn)
		return
	}
	handler.fn = fnValue
	fnType := fnValue.Type()
	nArgs := fnValue.Type().NumIn()
	handler.args = make([]reflect.Type, nArgs)
	if nArgs == 0 {
		err = errors.New("no arg exists")
		return
	}
	if t := fnType.In(0); t.Kind() != reflect.String {
		err = errors.New("first argument should be of type string")
		return
	} else {
		handler.args[0] = t
	}
	for i := 1; i < nArgs; i++ {
		/*
			if t.Kind() != reflect.Ptr {
				err = errors.New("function argument should be of type pointer")
				return
			}
		*/
		handler.args[i] = fnType.In(i)
	}
	eventHandlerCache.Lock()
	eventHandlerCache.cache[fnValue.Pointer()] = handler
	eventHandlerCache.Unlock()
	return
}

func (ee *EventEmitter) On(name string, fn interface{}) error {
	handler, err := genEventHandler(fn)
	if err != nil {
		return err
	}
	ee.mutex.Lock()
	defer ee.mutex.Unlock()
	ee.events[name] = append(ee.events[name], handler)
	return nil
}

func (ee *EventEmitter) AddListener(name string, fn interface{}) error {
	return ee.On(name, fn)
}

func (ee *EventEmitter) Once(name string, fn interface{}) error {
	handler, err := genEventHandler(fn)
	if err != nil {
		return err
	}
	ee.mutex.Lock()
	defer ee.mutex.Unlock()
	ee.eventsOnce[name] = append(ee.eventsOnce[name], handler)
	return nil
}

func (ee *EventEmitter) RemoveListener(name string, fn interface{}) {
	ee.mutex.Lock()
	defer ee.mutex.Unlock()
	for i, handler := range ee.events[name] {
		if handler.fn.Pointer() == reflect.ValueOf(fn).Pointer() {
			ee.events[name] = append(ee.events[name][0:i], ee.events[name][i+1:]...)
			break
		}
	}
	if len(ee.events[name]) == 0 {
		delete(ee.events, name)
	}
	for i, handler := range ee.eventsOnce[name] {
		if handler.fn.Pointer() == reflect.ValueOf(fn).Pointer() {
			ee.eventsOnce[name] = append(ee.eventsOnce[name][0:i], ee.eventsOnce[name][i+1:]...)
			break
		}
	}
	if len(ee.eventsOnce[name]) == 0 {
		delete(ee.eventsOnce, name)
	}
}

func (ee *EventEmitter) RemoveAllListeners(name string) {
	ee.mutex.Lock()
	defer ee.mutex.Unlock()
	// assign nil?
	delete(ee.events, name)
	delete(ee.eventsOnce, name)
}

func (ee *EventEmitter) fetchHandlers(name string) (handlers []*eventHandler) {
	ee.mutex.Lock()
	defer ee.mutex.Unlock()
	handlers = ee.eventsOnce[name]
	ee.eventsOnce[name] = nil
	delete(ee.eventsOnce, name)
	handlers = append(handlers, ee.events[name]...)
	return
}

func (ee *EventEmitter) emit(name string, callback func([]interface{}), args ...interface{}) {
	handlers := ee.fetchHandlers(name)
	callArgs := make([]reflect.Value, len(args)+1)
	callArgs[0] = reflect.ValueOf(name)
	for i, arg := range args {
		callArgs[i+1] = reflect.ValueOf(arg)
	}
	for _, handler := range handlers {
		go safeCall(handler.fn, callArgs, callback)
	}
}

func (ee *EventEmitter) emitRaw(name string, callback func([]interface{}), data []byte) {
	handlers := ee.fetchHandlers(name)
	for _, handler := range handlers {
		args := make([]interface{}, len(handler.args)-1)
		for i, arg := range handler.args[1:] {
			args[i] = reflect.New(arg).Interface()
		}
		err := json.Unmarshal(data, &args)
		if err != nil {
			log.Println(err)
			continue
		}
		callArgs := make([]reflect.Value, len(args)+1)
		callArgs[0] = reflect.ValueOf(name)
		for i, arg := range args {
			callArgs[i+1] = reflect.ValueOf(arg).Elem()
		}
		for _, handler := range handlers {
			go safeCall(handler.fn, callArgs, callback)
		}
	}
}

func safeCall(fn reflect.Value, args []reflect.Value, callback func([]interface{})) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()
	ret := fn.Call(args)
	if ret != nil {
		retArgs := make([]interface{}, len(ret))
		for i, arg := range ret {
			retArgs[i] = arg.Interface()
		}
		if callback != nil {
			callback(retArgs)
		}
	}
}
