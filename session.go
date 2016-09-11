package engineio

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/transport"
)

type session struct {
	params    base.ConnParameters
	manager   *manager
	closeOnce sync.Once

	upgradeLocker sync.RWMutex
	transport     string
	conn          base.Conn
}

func newSession(m *manager, t string, conn base.Conn, params base.ConnParameters) (*session, error) {
	params.SID = m.NewID()
	ret := &session{
		transport: t,
		conn:      conn,
		params:    params,
		manager:   m,
	}
	m.Add(ret)
	ret.setDeadline()

	go func() {
		w, err := ret.nextWriter(base.FrameString, base.OPEN)
		if err != nil {
			w.Close()
			ret.Close()
			return
		}
		if _, err := ret.params.WriteTo(w); err != nil {
			w.Close()
			ret.Close()
			return
		}
		if err := w.Close(); err != nil {
			ret.Close()
			return
		}
	}()

	return ret, nil
}

func (s *session) ID() string {
	return s.params.SID
}

func (s *session) Transport() string {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()
	return s.transport
}

func (s *session) Close() error {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()
	s.closeOnce.Do(func() {
		s.manager.Remove(s.params.SID)
	})
	return s.conn.Close()
}

func (s *session) NextReader() (FrameType, io.ReadCloser, error) {
	for {
		ft, pt, r, err := s.nextReader()
		if err != nil {
			return 0, nil, err
		}
		switch pt {
		case base.PING:
			err := func() error {
				w, err := s.nextWriter(ft, base.PONG)
				if err != nil {
					return err
				}
				io.Copy(w, r)
				return w.Close()
			}()
			r.Close()
			if err != nil {
				s.Close()
				return 0, nil, err
			}
			s.setDeadline()
		case base.CLOSE:
			r.Close()
			s.Close()
			return 0, nil, io.EOF
		case base.MESSAGE:
			return FrameType(ft), r, nil
		}
		r.Close()
	}
}

func (s *session) NextWriter(typ FrameType) (io.WriteCloser, error) {
	return s.nextWriter(base.FrameType(typ), base.MESSAGE)
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

func (s *session) nextReader() (base.FrameType, base.PacketType, io.ReadCloser, error) {
	var conn base.Conn
	var ft base.FrameType
	var pt base.PacketType
	var r io.Reader
	var err error
	for {
		fmt.Println("next reader lock")
		s.upgradeLocker.RLock()
		if conn == s.conn {
			if err != nil {
				fmt.Println("next reader unlock:", err)
				s.upgradeLocker.RUnlock()
				return 0, 0, nil, err
			}
			return ft, pt, newReader(r, &s.upgradeLocker), nil
		}
		conn = s.conn
		fmt.Println("next reader unlock")
		s.upgradeLocker.RUnlock()

		ft, pt, r, err = conn.NextReader()
	}
}

func (s *session) nextWriter(ft base.FrameType, pt base.PacketType) (io.WriteCloser, error) {
	fmt.Println("next writer lock", ft, pt)
	s.upgradeLocker.RLock()
	w, err := s.conn.NextWriter(ft, pt)
	if err != nil {
		fmt.Println("next writer unlock:", err)
		s.upgradeLocker.RUnlock()
		return nil, err
	}
	return newWriter(w, &s.upgradeLocker), nil
}

func (s *session) setDeadline() {
	deadline := time.Now().Add(s.params.PingTimeout)
	var conn base.Conn
	for {
		fmt.Println("set deadline lock")
		s.upgradeLocker.RLock()
		if conn == s.conn {
			fmt.Println("set deadline quit unlock")
			s.upgradeLocker.RUnlock()
			return
		}
		conn = s.conn
		fmt.Println("set deadline unlock")
		s.upgradeLocker.RUnlock()

		s.conn.SetReadDeadline(deadline)
		s.conn.SetWriteDeadline(deadline)
	}
}

func (s *session) upgrade(transport string, conn base.Conn) {
	go s.upgrading(transport, conn)
}

func (s *session) serveHTTP(w http.ResponseWriter, r *http.Request) {
	s.upgradeLocker.RLock()
	conn := s.conn
	s.upgradeLocker.RUnlock()

	if h, ok := conn.(http.Handler); ok {
		h.ServeHTTP(w, r)
	}
}

func (s *session) upgrading(t string, conn base.Conn) {
	defer fmt.Println("upgrading quit")
	deadline := time.Now().Add(s.params.PingTimeout)
	conn.SetReadDeadline(deadline)
	conn.SetWriteDeadline(deadline)

	fmt.Println("upgrading read ping")
	ft, pt, r, err := conn.NextReader()
	if err != nil {
		conn.Close()
		return
	}
	if pt != base.PING {
		conn.Close()
		return
	}

	func() {
		fmt.Println("upgrading lock")
		s.upgradeLocker.Lock()
		defer s.upgradeLocker.Unlock()
		defer fmt.Println("upgrading unlock")

		fmt.Println("upgrading write pong")
		w, err := conn.NextWriter(ft, base.PONG)
		if err != nil {
			conn.Close()
			return
		}
		if _, err := io.Copy(w, r); err != nil {
			conn.Close()
			return
		}
		if err := w.Close(); err != nil {
			conn.Close()
			return
		}

		fmt.Println("upgrading close old")
		s.conn.(transport.Pauser).Pause()

		fmt.Println("upgrading read upgrade")
		_, pt, _, err = conn.NextReader()
		if err != nil {
			conn.Close()
			return
		}
		if pt != base.UPGRADE {
			return
		}

		fmt.Println("upgrading switching")
		s.conn.Close()
		s.conn = conn
		s.transport = t
	}()
}
