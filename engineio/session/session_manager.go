package session

import (
	"sync"
)

type Manager struct {
	IDGenerator

	sessions map[string]*Session
	locker   sync.RWMutex
}

func NewManager(gen IDGenerator) *Manager {
	if gen == nil {
		gen = &DefaultIDGenerator{}
	}
	return &Manager{
		IDGenerator: gen,
		sessions:    make(map[string]*Session),
	}
}

func (m *Manager) Add(s *Session) {
	m.locker.Lock()
	defer m.locker.Unlock()

	m.sessions[s.ID()] = s
}

func (m *Manager) Get(sid string) *Session {
	m.locker.RLock()
	defer m.locker.RUnlock()

	return m.sessions[sid]
}

func (m *Manager) Remove(sid string) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if _, ok := m.sessions[sid]; !ok {
		return
	}
	delete(m.sessions, sid)
}

func (m *Manager) Count() int {
	m.locker.Lock()
	defer m.locker.Unlock()

	return len(m.sessions)
}
