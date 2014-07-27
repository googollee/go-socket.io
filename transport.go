package engineio

import (
	"io"
	"net/http"
	"sync"
)

// TransportCreateFunc is a function to create transport.
type transportCreateFunc func(req *http.Request) (transport, error)

// Transport is a trasport layer to connect server and client.
type transport interface {
	// Name returns the name of transport.
	Name() string
	// SetConn set the connection conn to transport.
	SetConn(conn Conn)
	// ServeHTTP handles the http request. It will call conn.onPacket when receive packet.
	ServeHTTP(http.ResponseWriter, *http.Request)
	// Close closes the transport
	Close() error
	// NextWriter returns packet writer. This function call should be synced.
	NextWriter(messageType MessageType, packetType packetType) (io.WriteCloser, error)
}

type transportMeta struct {
	creater         transportCreateFunc
	name            string
	handlesUpgrades bool
}

type transportsType struct {
	transports map[string]transportMeta
	locker     sync.RWMutex
}

var transports transportsType

func (t *transportsType) Register(name string, handlesUpgrades bool, creater transportCreateFunc) {
	t.locker.Lock()
	defer t.locker.Unlock()
	if t.transports == nil {
		t.transports = make(map[string]transportMeta)
	}
	t.transports[name] = transportMeta{
		creater:         creater,
		name:            name,
		handlesUpgrades: handlesUpgrades,
	}
}

func (t *transportsType) Names() []string {
	t.locker.RLock()
	defer t.locker.RUnlock()
	var ret []string
	for name, transport := range t.transports {
		if transport.handlesUpgrades {
			ret = append(ret, name)
		}
	}
	return ret
}

func (t *transportsType) GetCreater(name string) transportCreateFunc {
	t.locker.RLock()
	defer t.locker.RUnlock()
	ret, ok := t.transports[name]
	if !ok {
		return nil
	}
	return ret.creater
}

func (t *transportsType) GetUpgrade(name string) transportCreateFunc {
	t.locker.RLock()
	defer t.locker.RUnlock()
	ret, ok := t.transports[name]
	if !ok {
		return nil
	}
	if !ret.handlesUpgrades {
		return nil
	}
	return ret.creater
}
