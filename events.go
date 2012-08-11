package socketio

import (
	"reflect"
	"sync"
)

type Event struct {
	Name string  `json:"name"`
	Args argList `json:"args"`
}

type EventHandler func(event string, args []byte)

type EventEmitter struct {
	mutex      sync.Mutex
	events     map[string][]EventHandler
	eventsOnce map[string][]EventHandler
}

func NewEventEmitter() *EventEmitter {
	return &EventEmitter{events: make(map[string][]EventHandler)}
}

func (ee *EventEmitter) On(name string, fn EventHandler) {
	ee.mutex.Lock()
	defer ee.mutex.Unlock()
	ee.events[name] = append(ee.events[name], fn)
}

func (ee *EventEmitter) AddListener(name string, fn EventHandler) {
	ee.On(name, fn)
}

func (ee *EventEmitter) Once(name string, fn EventHandler) {
	ee.mutex.Lock()
	defer ee.mutex.Unlock()
	ee.eventsOnce[name] = append(ee.eventsOnce[name], fn)
}

func (ee *EventEmitter) RemoveListener(name string, fn EventHandler) {
	ee.mutex.Lock()
	defer ee.mutex.Unlock()
	for i, e := range ee.events[name] {
		if reflect.ValueOf(e).Pointer() == reflect.ValueOf(fn).Pointer() {
			ee.events[name] = append(ee.events[name][0:i], ee.events[name][i+1:]...)
			break
		}
	}
	if len(ee.events[name]) == 0 {
		delete(ee.events, name)
	}
	for i, e := range ee.eventsOnce[name] {
		if reflect.ValueOf(e).Pointer() == reflect.ValueOf(fn).Pointer() {
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
	ee.events[name] = nil
	delete(ee.events, name)
	ee.eventsOnce[name] = nil
	delete(ee.eventsOnce, name)
}

func (ee *EventEmitter) Listeners(name string) (handlers []EventHandler) {
	handlers = make([]EventHandler, len(ee.events[name])+len(ee.eventsOnce[name]))
	copy(handlers[0:len(ee.events)], ee.events[name])
	copy(handlers[len(ee.events):], ee.eventsOnce[name])
	return handlers
}

func (ee *EventEmitter) Emit(name string, args []byte) {
	ee.mutex.Lock()
	for _, fn := range ee.events[name] {
		go fn(name, args)
	}
	handlersOnce := ee.eventsOnce[name]
	ee.eventsOnce[name] = nil
	delete(ee.eventsOnce, name)
	ee.mutex.Unlock()

	for _, fn := range handlersOnce {
		go fn(name, args)
	}
}
