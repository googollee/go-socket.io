package engineio

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/googollee/go-engine.io/base"
)

type session struct {
	id        string
	manager   *manager
	closeOnce sync.Once

	upgradeLocker sync.RWMutex
	transport     string
	conn          base.Conn
	params        base.ConnParameters

	writeLocker sync.Mutex
}

func newSession(m *manager, t string, conn base.Conn, params base.ConnParameters) *session {
	ret := &session{
		id:        m.NewID(),
		transport: t,
		conn:      conn,
		params:    params,
		manager:   m,
	}
	m.Add(ret)
	return ret
}

func (s *session) ID() string {
	return s.id
}

func (s *session) Transport() string {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()
	return s.transport
}

func (s *session) Close() error {
	s.closeOnce.Do(func() {
		s.manager.Remove(s.id)
	})
	return s.conn.Close()
}

func (s *session) NextReader() (FrameType, io.Reader, error) {
	for {
		s.upgradeLocker.RLock()
		ft, pt, r, err := s.conn.NextReader()
		s.upgradeLocker.RUnlock()
		if err != nil {
			return 0, nil, err
		}
		switch pt {
		case base.PING:
			err := func() error {
				s.writeLocker.Lock()
				defer s.writeLocker.Unlock()

				s.upgradeLocker.RLock()
				w, err := s.conn.NextWriter(ft, base.PONG)
				s.upgradeLocker.RUnlock()
				if err != nil {
					return err
				}
				io.Copy(w, r)
				return w.Close()
			}()
			if err != nil {
				return 0, nil, err
			}
		case base.CLOSE:
			s.Close()
			return 0, nil, io.EOF
		case base.NOOP:
		case base.MESSAGE:
			return FrameType(ft), r, nil
		case base.OPEN:
			fallthrough
		default:
		}
	}
}

func (s *session) NextWriter(typ FrameType) (io.WriteCloser, error) {
	s.writeLocker.Lock()
	defer s.writeLocker.Unlock()

	s.upgradeLocker.RLock()
	w, err := s.conn.NextWriter(base.FrameType(typ), base.MESSAGE)
	s.upgradeLocker.RUnlock()
	return w, err
}

func (s *session) LocalAddr() string {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()
	return s.conn.LocalAddr()
}

func (s *session) RemoteAddr() string {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()
	return s.conn.RemoteAddr()
}

func (s *session) RemoteHeader() http.Header {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()
	return s.conn.RemoteHeader()
}

func (s *session) upgrade(params base.ConnParameters, transport string, conn base.Conn) {
	go s.upgrading(params, transport, conn)
}

func (s *session) serveHTTP(w http.ResponseWriter, r *http.Request) {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()
	if h, ok := s.conn.(http.Handler); ok {
		h.ServeHTTP(w, r)
	}
}

func (s *session) upgrading(params base.ConnParameters, transport string, conn base.Conn) {
	conn.SetReadDeadline(time.Now().Add(params.PingTimeout))
	ft, pt, r, err := conn.NextReader()
	if err != nil {
		return
	}
	if pt != base.PING {
		return
	}
	conn.SetWriteDeadline(time.Now().Add(params.PingTimeout))
	w, err := conn.NextWriter(ft, base.PONG)
	if err != nil {
		return
	}
	if _, err := io.Copy(w, r); err != nil {
		return
	}
	if err := w.Close(); err != nil {
		return
	}
	_, pt, _, err = conn.NextReader()
	if err != nil {
		return
	}
	if pt != base.UPGRADE {
		return
	}

	s.conn.Close()
	s.upgradeLocker.Lock()
	defer s.upgradeLocker.Unlock()
	s.conn = conn
	s.params = params
	s.transport = transport
}
