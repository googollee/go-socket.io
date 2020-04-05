package engineio

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/googollee/go-socket.io/connection/base"
	"github.com/googollee/go-socket.io/connection/payload"
	"github.com/googollee/go-socket.io/connection/transport"
)

type session struct {
	params    base.ConnParameters
	manager   *manager
	closeOnce sync.Once
	context   interface{}

	upgradeLocker sync.RWMutex
	transport     string
	conn          base.Conn
}

func newSession(m *manager, t string, conn base.Conn, params base.ConnParameters) (*session, error) {
	params.SID = m.NewID()
	ses := &session{
		transport: t,
		conn:      conn,
		params:    params,
		manager:   m,
	}

	if err := ses.setDeadline(); err != nil {
		ses.Close()
		return nil, err
	}

	m.Add(ses)

	return ses, nil
}

func (s *session) SetContext(v interface{}) {
	s.context = v
}

func (s *session) Context() interface{} {
	return s.context
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

// NextReader attempts to obtain a ReadCloser from the session's connection.
// When finished writing, the caller MUST Close the ReadCloser to unlock the
// connection's FramerReader.
func (s *session) NextReader() (FrameType, io.ReadCloser, error) {
	for {
		ft, pt, r, err := s.nextReader()
		if err != nil {
			s.Close()
			return 0, nil, err
		}
		switch pt {
		case base.PING:
			// Respond to a ping with a pong.
			err := func() error {
				w, err := s.nextWriter(ft, base.PONG)
				if err != nil {
					return err
				}
				// echo
				_, err = io.Copy(w, r)
				w.Close() // unlocks the wrapped connection's FrameWriter
				r.Close() // unlocks the wrapped connection's FrameReader
				return err
			}()
			if err != nil {
				s.Close()
				return 0, nil, err
			}
			// Read another frame.
			if err := s.setDeadline(); err != nil {
				s.Close()
				return 0, nil, err
			}
		case base.CLOSE:
			r.Close() // unlocks the wrapped connection's FrameReader
			s.Close()
			return 0, nil, io.EOF
		case base.MESSAGE:
			// Caller must Close the ReadCloser to unlock the connection's
			// FrameReader when finished reading.
			return FrameType(ft), r, nil
		default:
			// Unknown packet type. Close reader and try again.
			r.Close()
		}
	}
}

// NextWriter attempts to obtain a WriteCloser from the session's connection.
// When finished writing, the caller MUST Close the WriteCloser to unlock the
// connection's FrameWriter.
func (s *session) NextWriter(typ FrameType) (io.WriteCloser, error) {
	return s.nextWriter(base.FrameType(typ), base.MESSAGE)
}

func (s *session) URL() url.URL {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()
	return s.conn.URL()
}

func (s *session) LocalAddr() net.Addr {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()
	return s.conn.LocalAddr()
}

func (s *session) RemoteAddr() net.Addr {
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
	for {
		s.upgradeLocker.RLock()
		conn := s.conn
		s.upgradeLocker.RUnlock()
		ft, pt, r, err := conn.NextReader()
		if err != nil {
			if op, ok := err.(payload.Error); ok && op.Temporary() {
				continue
			}
			return 0, 0, nil, err
		}
		return ft, pt, r, nil
	}
}

func (s *session) nextWriter(ft base.FrameType, pt base.PacketType) (io.WriteCloser, error) {
	for {
		s.upgradeLocker.RLock()
		conn := s.conn
		s.upgradeLocker.RUnlock()
		w, err := conn.NextWriter(ft, pt)
		if err != nil {
			if op, ok := err.(payload.Error); ok && op.Temporary() {
				continue
			}
			return nil, err
		}
		// Caller must Close the WriteCloser to unlock the connection's
		// FrameWriter when finished writing.
		return w, nil
	}
}

func (s *session) setDeadline() error {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()

	deadline := time.Now().Add(s.params.PingTimeout)

	err := s.conn.SetReadDeadline(deadline)
	if err != nil {
		return err
	}
	return s.conn.SetWriteDeadline(deadline)
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
	// Read a ping from the client.
	err := conn.SetReadDeadline(time.Now().Add(s.params.PingTimeout))
	if err != nil {
		conn.Close()
		return
	}

	ft, pt, r, err := conn.NextReader()
	if err != nil {
		conn.Close()
		return
	}
	if pt != base.PING {
		r.Close()
		conn.Close()
		return
	}
	// Wait to close the reader until after data is read and echoed in the reply.

	// Sent a pong in reply.
	err = conn.SetWriteDeadline(time.Now().Add(s.params.PingTimeout))
	if err != nil {
		r.Close()
		conn.Close()
		return
	}

	w, err := conn.NextWriter(ft, base.PONG)
	if err != nil {
		r.Close()
		conn.Close()
		return
	}
	// echo
	if _, err = io.Copy(w, r); err != nil {
		w.Close()
		r.Close()
		conn.Close()
		return
	}
	if err = r.Close(); err != nil {
		w.Close()
		conn.Close()
		return
	}
	if err = w.Close(); err != nil {
		conn.Close()
		return
	}

	// Pause the old connection.
	s.upgradeLocker.RLock()
	old := s.conn
	s.upgradeLocker.RUnlock()
	p, ok := old.(transport.Pauser)
	if !ok {
		// old transport doesn't support upgrading
		conn.Close()
		return
	}
	p.Pause()
	// Prepare to resume the connection if upgrade fails.
	defer func() {
		if p != nil {
			p.Resume()
		}
	}()

	// Check for upgrade packet from the client.
	_, pt, r, err = conn.NextReader()
	if err != nil {
		conn.Close()
		return
	}
	if pt != base.UPGRADE {
		r.Close()
		conn.Close()
		return
	}
	if err = r.Close(); err != nil {
		conn.Close()
		return
	}

	// Successful upgrade.
	s.upgradeLocker.Lock()
	s.conn = conn
	s.transport = t
	s.upgradeLocker.Unlock()
	p = nil

	old.Close()
}
