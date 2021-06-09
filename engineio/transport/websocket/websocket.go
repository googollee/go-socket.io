package websocket

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/transport"
)

type ws struct {
	pingTimeout time.Duration
	allocator   transport.BufferAllocator
	callbacks   transport.Callbacks

	conn *websocket.Conn
}

func newWebsocket(pingTimeout time.Duration, alloc transport.BufferAllocator, callbacks transport.Callbacks) transport.Transport {
	return &ws{
		pingTimeout: pingTimeout,
		allocator:   alloc,
		callbacks:   callbacks,
	}
}

func (s *ws) Name() string {
	return string(transport.Websocket)
}

func (s *ws) Close() error {
	return s.conn.Close()
}

func (s *ws) SendFrame(ft frame.Type) (io.WriteCloser, error) {
	typ, ok := toMessageType(ft)
	if !ok {
		return nil, errors.New("invalid frame type")
	}

	return s.conn.NextWriter(typ)
}

func (s *ws) PrepareHTTP(w http.ResponseWriter, r *http.Request) error {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	s.conn = conn
	return nil
}

func (s *ws) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.conn == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid transport"))
		return
	}

	defer s.conn.Close()

	nextPing := time.Now().Add(s.pingTimeout)
	for {
		if err := s.conn.SetReadDeadline(nextPing); err != nil {
			s.callbacks.OnError(s, err)
			return
		}

		typ, rd, err := s.conn.NextReader()
		if err == websocket.ErrReadLimit {
			s.callbacks.OnPingTimeout(s)
			nextPing = time.Now().Add(s.pingTimeout)
			continue
		}
		if err != nil {
			s.callbacks.OnError(s, err)
			continue
		}

		ft, ok := toFrameType(typ)
		if !ok {
			s.callbacks.OnError(s, fmt.Errorf("invalid websocket message type: %d", typ))
			continue
		}
		s.callbacks.OnFrame(s, r, ft, rd)
	}
}

func toFrameType(t int) (frame.Type, bool) {
	switch t {
	case websocket.BinaryMessage:
		return frame.Binary, true
	case websocket.TextMessage:
		return frame.Text, true
	}

	return frame.Text, false
}

func toMessageType(ft frame.Type) (int, bool) {
	switch ft {
	case frame.Text:
		return websocket.TextMessage, true
	case frame.Binary:
		return websocket.BinaryMessage, true
	}

	return 0, false
}
