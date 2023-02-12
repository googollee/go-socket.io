package transport

import (
	"net/http"

	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

type ConnectGenerator func(w http.ResponseWriter, r *http.Request) (Conn, error)

// Manager is a manager of transports.
type Manager struct {
	order      []Type
	transports map[Type]ConnectGenerator
}

// NewManager creates a new manager.
func NewManager(transports []Type) *Manager {
	mapping := make(map[string]ConnectGenerator)
	names := make([]string, len(transports))

	for i, t := range transports {
		names[i] = t
		mapping[t] = getGenerator(t)
	}

	return &Manager{
		order:      names,
		transports: mapping,
	}
}

// UpgradeFrom returns a name list of transports which can upgrade from given name.
func (m *Manager) UpgradeFrom(name Type) []string {
	for i, n := range m.order {
		if n == name {
			return m.order[i+1:]
		}
	}

	return nil
}

// Get returns the transport with given name.
func (m *Manager) Get(name Type) (ConnectGenerator, bool) {
	t, ok := m.transports[name]
	return t, ok
}

func getGenerator(transport Type) ConnectGenerator {
	switch transport {
	case Polling:
		return polling.New
	case Websocket:
		return websocket.New
	}

	return nil
}
