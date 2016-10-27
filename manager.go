package engineio

import (
	"strconv"
	"sync"
	"sync/atomic"
)

// SessionIDGenerator generates new session id. Default behavior is simple
// increasing number.
// If you need custom session id, for example using local ip as perfix, you can
// implement SessionIDGenerator and save in Configure. Engine.io will use custom
// one to generate new session id.
type SessionIDGenerator interface {
	NewID() string
}

type defaultIDGenerator struct {
	nextID uint64
}

func (g *defaultIDGenerator) NewID() string {
	id := atomic.AddUint64(&g.nextID, 1)
	return strconv.FormatUint(id, 36)
}

type manager struct {
	SessionIDGenerator

	s      map[string]*session
	locker sync.RWMutex
}

func newManager(g SessionIDGenerator) *manager {
	if g == nil {
		g = &defaultIDGenerator{}
	}
	return &manager{
		SessionIDGenerator: g,
		s:                  make(map[string]*session),
	}
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
