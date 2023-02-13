package transport

import (
	"errors"
	"net/http"

	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

// Manager is a manager of transports.
type Manager struct {
	order []Type
}

// NewManager creates a new manager.
func NewManager(transports []Type) *Manager {
	names := make([]string, len(transports))

	for i, t := range transports {
		names[i] = t
	}

	return &Manager{
		order: names,
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

func (m *Manager) CreateConnection(transport Type, w http.ResponseWriter, r *http.Request) (Conn, error) {
	switch transport {
	case Polling:
		return polling.New(w, r)
	case Websocket:
		return websocket.New(w, r)
	}

	return nil, errors.New("invalid transport type")
}
