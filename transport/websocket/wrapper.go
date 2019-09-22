package websocket

import (
	"io"
	"io/ioutil"
	"sync"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/transport"
	"github.com/gorilla/websocket"
)

type wrapper struct {
	*websocket.Conn
	writeLocker *sync.Mutex
	readLocker  *sync.Mutex
}

func newWrapper(conn *websocket.Conn) wrapper {
	return wrapper{
		Conn:        conn,
		writeLocker: new(sync.Mutex),
		readLocker:  new(sync.Mutex),
	}
}

func (w wrapper) NextReader() (base.FrameType, io.ReadCloser, error) {
	w.readLocker.Lock()
	typ, r, err := w.Conn.NextReader()
	// The wrapper remains locked until the returned ReadCloser is Closed.
	if err != nil {
		w.readLocker.Unlock()
		return 0, nil, err
	}
	rc := rcWrapper{w.readLocker, r}
	switch typ {
	case websocket.TextMessage:
		return base.FrameString, rc, nil
	case websocket.BinaryMessage:
		return base.FrameBinary, rc, nil
	}
	return 0, nil, transport.ErrInvalidFrame
}

type rcWrapper struct {
	l *sync.Mutex
	io.Reader
}

func (r rcWrapper) Close() error {
	io.Copy(ioutil.Discard, r) // reader may be closed, ignore error
	r.l.Unlock()
	return nil
}

func (w wrapper) NextWriter(typ base.FrameType) (io.WriteCloser, error) {
	var t int
	switch typ {
	case base.FrameString:
		t = websocket.TextMessage
	case base.FrameBinary:
		t = websocket.BinaryMessage
	default:
		return nil, transport.ErrInvalidFrame
	}

	w.writeLocker.Lock()
	writer, err := w.Conn.NextWriter(t)
	// The wrapper remains locked until the returned WriteCloser is Closed.
	if err != nil {
		w.writeLocker.Unlock()
		return nil, err
	}

	return wcWrapper{w.writeLocker, writer}, nil
}

type wcWrapper struct {
	l *sync.Mutex
	io.WriteCloser
}

func (w wcWrapper) Close() error {
	defer w.l.Unlock()
	return w.WriteCloser.Close()
}
