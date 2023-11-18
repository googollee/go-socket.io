package transport

import (
	"net/http"
	"net/url"
)

// Transport is a transport which can creates base.Conn
type Transport interface {
	Name() string
	Accept(w http.ResponseWriter, r *http.Request) (Conn, error)
	Dial(u *url.URL, requestHeader http.Header) (Conn, error)
}

// Manager is a manager of transports.
type Manager struct {
	order      []string
	transports map[string]Transport
}

// NewManager creates a new manager.
func NewManager(transports []Transport) *Manager {
	tranMap := make(map[string]Transport)
	names := make([]string, len(transports))
	for i, t := range transports {
		names[i] = t.Name()
		tranMap[t.Name()] = t
	}

	return &Manager{
		order:      names,
		transports: tranMap,
	}
}

// UpgradeFrom returns a name list of transports which can upgrade from given
// name.
func (m *Manager) UpgradeFrom(name string) []string {
	for i, n := range m.order {
		if n == name {
			return m.order[i+1:]
		}
	}
	return nil
}

// Get returns the transport with given name.
func (m *Manager) Get(name string) (Transport, bool) {
	t, ok := m.transports[name]
	return t, ok
}
