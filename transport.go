package engineio

import (
	"io"
	"net/http"
)

// TransportCreateFunc is a function to create transport.
type TransportCreateFunc func(req *http.Request) (Transport, error)

// Transport is a trasport layer to connect server and client.
type Transport interface {
	// Name returns the name of transport.
	Name() string
	// SetConn set the connection conn to transport.
	SetConn(conn Conn)
	// ServeHTTP handles the http request. It will call conn.onPacket when receive packet.
	ServeHTTP(http.ResponseWriter, *http.Request)
	// Close closes the transport
	Close() error
	// NextWriter returns packet writer. This function call should be synced.
	NextWriter(messageType MessageType, packetType PacketType) (io.WriteCloser, error)
}

type transportMeta struct {
	creater         TransportCreateFunc
	name            string
	handlesUpgrades bool
}

type transportsType map[string]transportMeta

var transports = make(transportsType)

// RegisterTransport registers a transport with name and whether can handle upgrades.
func RegisterTransport(name string, handlesUpgrades bool, creater TransportCreateFunc) {
	transports[name] = transportMeta{
		creater:         creater,
		name:            name,
		handlesUpgrades: handlesUpgrades,
	}
}

func getUpgradesHandlers() []string {
	var ret []string
	for name, transport := range transports {
		if transport.handlesUpgrades {
			ret = append(ret, name)
		}
	}
	return ret
}

func getTransportCreater(name string) TransportCreateFunc {
	ret, ok := transports[name]
	if !ok {
		return nil
	}
	return ret.creater
}

func getTransportUpgrade(name string) TransportCreateFunc {
	ret, ok := transports[name]
	if !ok {
		return nil
	}
	if !ret.handlesUpgrades {
		return nil
	}
	return ret.creater
}
