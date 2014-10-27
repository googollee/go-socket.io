package engineio

import (
	"sync"
)

type serverSessions struct {
	sessions map[string]*serverConn
	locker   sync.RWMutex
}

func newServerSessions() *serverSessions {
	return &serverSessions{
		sessions: make(map[string]*serverConn),
	}
}

func (s *serverSessions) Get(id string) *serverConn {
	s.locker.RLock()
	defer s.locker.RUnlock()

	ret, ok := s.sessions[id]
	if !ok {
		return nil
	}
	return ret
}

func (s *serverSessions) Set(id string, serverConn *serverConn) {
	s.locker.Lock()
	defer s.locker.Unlock()

	s.sessions[id] = serverConn
}

func (s *serverSessions) Remove(id string) {
	s.locker.Lock()
	defer s.locker.Unlock()

	delete(s.sessions, id)
}
