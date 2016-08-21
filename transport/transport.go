package transport

import (
	"net/http"

	"github.com/googollee/go-engine.io/base"
)

// Transport is a transport which can creates base.Conn
type Transport interface {
	ServeHTTP(conn base.Conn, header http.Header, w http.ResponseWriter, r *http.Request)
	ConnChan() <-chan base.Conn
}

// Manager is a manager of transports.
type Manager map[string]Transport

// NewManager creates a new manager.
func NewManager() Manager {
	return make(Manager)
}

// OtherNames returns a name list of transports except given name.
func (m Manager) OtherNames(name string) []string {
	ret := make([]string, 0, len(m))
	for k := range m {
		if k == name {
			continue
		}
		ret = append(ret, k)
	}
	return ret
}

// Get returns the transport with given name.
func (m Manager) Get(name string) Transport {
	return m[name]
}

// Register registers a transport t with name.
func (m Manager) Register(name string, t Transport) {
	if t == nil {
		panic("can't register nil transport")
	}
	m[name] = t
}
