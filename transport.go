package engineio

import (
	"io"
	"net/http"
	"time"
)

type TransportCreateFunc func(req *http.Request, pingInterval, pingTimeout time.Duration) (Transport, error)

type Transport interface {
	Name() string
	SetSocket(socket Conn)
	HandlesUpgrades() bool
	SupportsFraming() bool
	ServeHTTP(http.ResponseWriter, *http.Request)
	Close() error
	Upgraded() error
	NextWriter(messageType MessageType, packetType PacketType) (io.WriteCloser, error)
}

type transportMeta struct {
	creater         TransportCreateFunc
	name            string
	handlesUpgrades bool
	supportsFraming bool
}

type transportsType map[string]transportMeta

var transports = make(transportsType)

func RegisterTransport(name string, handlesUpgrades, supportsFraming bool, creater TransportCreateFunc) {
	transports[name] = transportMeta{
		creater:         creater,
		name:            name,
		handlesUpgrades: handlesUpgrades,
		supportsFraming: supportsFraming,
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
