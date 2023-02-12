package transport

// Manager is a manager of transports.
type Manager struct {
	order      []Type
	transports map[string]Conn
}

// NewManager creates a new manager.
func NewManager(transports []Type) *Manager {
	tranMap := make(map[string]Conn)
	names := make([]string, len(transports))

	for i, t := range transports {
		transportString := t.String()

		names[i] = transportString
		tranMap[transportString] = t
	}

	return &Manager{
		order:      names,
		transports: tranMap,
	}
}

// UpgradeFrom returns a name list of transports which can upgrade from given name.
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
