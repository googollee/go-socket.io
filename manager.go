package engineio

import (
	"strconv"
	"sync"
	"sync/atomic"
)

type manager struct {
	s      map[string]*session
	nextID uint64
	locker sync.RWMutex
}

func newManager() *manager {
	return &manager{
		s: make(map[string]*session),
	}
}

func (m *manager) NewID() string {
	id := atomic.AddUint64(&m.nextID, 1)
	return strconv.FormatUint(id, 36)
}

func (m *manager) Add(s *session) {
	m.locker.Lock()
	defer m.locker.Unlock()
	m.s[s.ID()] = s
}

func (m *manager) Get(sid string) *session {
	m.locker.RLock()
	defer m.locker.RUnlock()
	return m.s[sid]
}

func (m *manager) Remove(sid string) {
	m.locker.Lock()
	defer m.locker.Unlock()
	if _, ok := m.s[sid]; !ok {
		return
	}
	delete(m.s, sid)
}
