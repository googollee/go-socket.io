package engineio

import (
	"fmt"
	"io"
	"net/http"
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

type transportsType map[string]transportMeta

var allTransports transportsType

func registerTransport(name string, handlesUpgrades bool, creater transportCreateFunc) {
	if allTransports == nil {
		allTransports = make(transportsType)
	}
	allTransports[name] = transportMeta{
		creater:         creater,
		name:            name,
		handlesUpgrades: handlesUpgrades,
	}
}

func newTransportsType(names []string) (transportsType, error) {
	ret := make(transportsType)
	if names == nil {
		return allTransports, nil
	}
	for _, name := range names {
		t, ok := allTransports[name]
		if !ok {
			return nil, fmt.Errorf("invalid transport name %s", name)
		}
		ret[name] = t
	}
	return ret, nil
}

func (t transportsType) Upgrades() []string {
	var ret []string
	for name, transport := range t {
		if transport.handlesUpgrades {
			ret = append(ret, name)
		}
	}
	return ret
}

func (t transportsType) GetCreater(name string) transportCreateFunc {
	ret, ok := t[name]
	if !ok {
		return nil
	}
	return ret.creater
}

func (t transportsType) GetUpgrade(name string) transportCreateFunc {
	ret, ok := t[name]
	if !ok {
		return nil
	}
	if !ret.handlesUpgrades {
		return nil
	}
	return ret.creater
}
