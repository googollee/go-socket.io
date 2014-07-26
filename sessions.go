package engineio

import (
	"sync"
)

type sessions struct {
	sessions map[string]*conn
	locker   sync.RWMutex
}

func newSessions() *sessions {
	return &sessions{
		sessions: make(map[string]*conn),
	}
}

func (s *sessions) Get(id string) *conn {
	s.locker.RLock()
	defer s.locker.RUnlock()

	ret, ok := s.sessions[id]
	if !ok {
		return nil
	}
	return ret
}

func (s *sessions) Set(id string, conn *conn) {
	s.locker.Lock()
	defer s.locker.Unlock()

	s.sessions[id] = conn
}

func (s *sessions) Remove(id string) {
	s.locker.Lock()
	defer s.locker.Unlock()

	delete(s.sessions, id)
}
