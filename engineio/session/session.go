package session

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/googollee/go-socket.io/engineio/packet"
	"github.com/googollee/go-socket.io/engineio/payload"
	"github.com/googollee/go-socket.io/engineio/transport"
)

// Pauser is connection which can be paused and resumes.
type Pauser interface {
	Pause()
	Resume()
}

type Session struct {
	conn      transport.Conn
	params    transport.ConnParameters
	transport string

	context interface{}

	upgradeLocker sync.RWMutex
}

func New(conn transport.Conn, sid, transport string, params transport.ConnParameters) (*Session, error) {
	params.SID = sid

	ses := &Session{
		transport: transport,
		conn:      conn,
		params:    params,
	}

	if err := ses.setDeadline(); err != nil {
		ses.Close()
		return nil, err
	}

	return ses, nil
}

func (s *Session) SetContext(v interface{}) {
	s.context = v
}

func (s *Session) Context() interface{} {
	return s.context
}

func (s *Session) ID() string {
	return s.params.SID
}

func (s *Session) Transport() string {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()

	return s.transport
}

func (s *Session) Close() error {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()

	return s.conn.Close()
}

// NextReader attempts to obtain a ReadCloser from the session's connection.
// When finished writing, the caller MUST Close the ReadCloser to unlock the
// connection's FramerReader.
func (s *Session) NextReader() (FrameType, io.ReadCloser, error) {
	for {
		ft, pt, r, err := s.nextReader()
		if err != nil {
			s.Close()
			return 0, nil, err
		}

		switch pt {
		case packet.PING:
			// Respond to a ping with a pong.
			err := func() error {
				w, err := s.nextWriter(ft, packet.PONG)
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

		case packet.CLOSE:
			r.Close() // unlocks the wrapped connection's FrameReader
			s.Close()
			return 0, nil, io.EOF

		case packet.MESSAGE:
			// Caller must Close the ReadCloser to unlock the connection's
			// FrameReader when finished reading.
			return FrameType(ft), r, nil

		default:
			// Unknown packet type. Close reader and try again.
			r.Close()
		}
	}
}

func (s *Session) URL() url.URL {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()

	return s.conn.URL()
}

func (s *Session) LocalAddr() net.Addr {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()

	return s.conn.LocalAddr()
}

func (s *Session) RemoteAddr() net.Addr {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()

	return s.conn.RemoteAddr()
}

func (s *Session) RemoteHeader() http.Header {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()

	return s.conn.RemoteHeader()
}

// NextWriter attempts to obtain a WriteCloser from the session's connection.
// When finished writing, the caller MUST Close the WriteCloser to unlock the
// connection's FrameWriter.
func (s *Session) NextWriter(typ FrameType) (io.WriteCloser, error) {
	return s.nextWriter(packet.FrameType(typ), packet.MESSAGE)
}

func (s *Session) Upgrade(transport string, conn transport.Conn) {
	go s.upgrading(transport, conn)
}

func (s *Session) InitSession() error {
	w, err := s.nextWriter(packet.FrameString, packet.OPEN)
	if err != nil {
		s.Close()
		return err
	}

	if _, err := s.params.WriteTo(w); err != nil {
		w.Close()
		s.Close()
		return err
	}

	if err := w.Close(); err != nil {
		s.Close()
		return err
	}

	return nil
}

func (s *Session) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.upgradeLocker.RLock()
	conn := s.conn
	s.upgradeLocker.RUnlock()

	if h, ok := conn.(http.Handler); ok {
		h.ServeHTTP(w, r)
	}
}

func (s *Session) nextReader() (packet.FrameType, packet.PacketType, io.ReadCloser, error) {
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

func (s *Session) nextWriter(ft packet.FrameType, pt packet.PacketType) (io.WriteCloser, error) {
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

func (s *Session) setDeadline() error {
	s.upgradeLocker.RLock()
	defer s.upgradeLocker.RUnlock()

	deadline := time.Now().Add(s.params.PingTimeout)

	err := s.conn.SetReadDeadline(deadline)
	if err != nil {
		return err
	}

	return s.conn.SetWriteDeadline(deadline)
}

func (s *Session) upgrading(t string, conn transport.Conn) {
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
	if pt != packet.PING {
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

	w, err := conn.NextWriter(ft, packet.PONG)
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

	p, ok := old.(Pauser)
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

	if pt != packet.UPGRADE {
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
