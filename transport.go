package socketio

import (
	"net/http"
	"sync"
)

type Transport interface {
	Name() string
	OnData(http.ResponseWriter, *http.Request)
	Send([]byte) error
	Close()
}

type newTransportFunc func(session *Session, heartbeatTimeout int) Transport

var (
	DefaultTransports = NewTransportManager()
)

type TransportManager struct {
	mutex      sync.RWMutex
	transports map[string]newTransportFunc
}

func NewTransportManager() *TransportManager {
	return &TransportManager{transports: make(map[string]newTransportFunc)}
}

func (tm *TransportManager) RegisterTransport(name string, f newTransportFunc) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	tm.transports[name] = f
}

func (tm *TransportManager) GetTransportNames() (names []string) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	names = make([]string, 0, len(tm.transports))
	for k, _ := range tm.transports {
		names = append(names, k)
	}
	return
}

func (tm *TransportManager) Get(name string, session *Session, heartbeatTimeout int) Transport {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	f, ok := tm.transports[name]
	if !ok {
		return nil
	}
	return f(session, heartbeatTimeout)
}
