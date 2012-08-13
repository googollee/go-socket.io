package socketio

import (
	"net/http"
	"sync"
)

type Transport interface {
	Name() string
	New(*Session) Transport
	OnData(http.ResponseWriter, *http.Request)
	Send([]byte)
	Close()
}

var (
	DefaultTransports = NewTransportManager()
)

type TransportManager struct {
	mutex      sync.RWMutex
	transports map[string]Transport
}

func NewTransportManager() *TransportManager {
	return &TransportManager{transports: make(map[string]Transport)}
}
func (tm *TransportManager) RegisterTransport(transport Transport) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	tm.transports[transport.Name()] = transport
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

func (tm *TransportManager) Get(name string) Transport {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	return tm.transports[name]
}
