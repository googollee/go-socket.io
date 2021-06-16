package websocket

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/transport"
)

type ws struct {
	pingTimeout time.Duration
	allocator   transport.BufferAllocator
	callbacks   transport.Callbacks

	conn         *websocket.Conn
	writerLocker chan struct{}
}

func newWebsocket(pingTimeout time.Duration, alloc transport.BufferAllocator, callbacks transport.Callbacks) transport.Transport {
	return &ws{
		pingTimeout:  pingTimeout,
		allocator:    alloc,
		callbacks:    callbacks,
		writerLocker: make(chan struct{}, 1),
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

	select {
	case s.writerLocker <- struct{}{}:
	default:
		return nil, errors.New("non-closed frame")
	}

	ret, err := s.conn.NextWriter(typ)
	if err != nil {
		return nil, err
	}

	return &writer{
		WriteCloser: ret,
		locker:      s.writerLocker,
	}, nil
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
		if _, err := w.Write([]byte("invalid transport")); err != nil {
			s.callbacks.OnError(s, err)
		}
		return
	}

	defer s.conn.Close()

	pingTicker := time.NewTicker(s.pingTimeout)
	closeChan := make(chan struct{})
	var wg sync.WaitGroup
	defer func() {
		pingTicker.Stop()
		close(closeChan)
		wg.Wait()
	}()

	wg.Add(1)
	go func() {
		for {
			select {
			case <-pingTicker.C:
				s.callbacks.OnPingTimeout(s)
			case <-closeChan:
				return
			}
		}
	}()

	for {
		typ, rd, err := s.conn.NextReader()
		if err != nil {
			s.callbacks.OnError(s, err)
			continue
		}

		ft, ok := toFrameType(typ)
		if !ok {
			continue
		}
		if err := s.callbacks.OnFrame(s, r, ft, rd); err != nil {
			code := http.StatusInternalServerError
			if he, ok := err.(transport.HTTPError); ok {
				code = he.Code()
			}
			s.responseHTTP(w, r, code, err.Error())
			return
		}
	}
}

func (s *ws) responseHTTP(w http.ResponseWriter, r *http.Request, code int, msg string) {
	w.WriteHeader(code)
	data := []byte(msg)
	for len(data) > 0 {
		n, err := w.Write([]byte(msg))
		data = data[n:]
		if err != nil {
			s.callbacks.OnError(s, err)
			return
		}
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

type writer struct {
	io.WriteCloser
	locker chan struct{}
}

func (w *writer) Close() error {
	select {
	case <-w.locker:
	default:
	}

	return w.WriteCloser.Close()
}
