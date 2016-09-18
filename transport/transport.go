package transport

import (
	"net/http"

	"github.com/googollee/go-engine.io/base"
)

type Opener interface {
	Open(url string, requestHeader http.Header) (base.ConnParameters, error)
}

type Pauser interface {
	Pause()
	Resume()
}

// HTTPError is error which has http response code
type HTTPError interface {
	Code() int
}

// Transport is a transport which can creates base.Conn
type Transport interface {
	Name() string
}

// Transport is a transport which can creates base.Conn
type Server interface {
	Accept(w http.ResponseWriter, r *http.Request) (base.Conn, error)
}

type UpgradeClient interface {
	Dial(url string, requestHeader http.Header) (base.Conn, error)
}

type OpenClient interface {
	Open(url string, requestHeader http.Header) (base.Conn, base.ConnParameters, error)
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
func (m *Manager) Get(name string) Transport {
	return m.transports[name]
}
