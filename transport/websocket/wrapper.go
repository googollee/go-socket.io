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
}

func newWrapper(conn *websocket.Conn) wrapper {
	return wrapper{
		Conn:        conn,
		writeLocker: new(sync.Mutex),
	}
}

func (w wrapper) NextReader() (base.FrameType, io.ReadCloser, error) {
	typ, r, err := w.Conn.NextReader()
	if err != nil {
		return 0, nil, err
	}
	ret := ioutil.NopCloser(r)
	switch typ {
	case websocket.TextMessage:
		return base.FrameString, ret, nil
	case websocket.BinaryMessage:
		return base.FrameBinary, ret, nil
	}
	return 0, nil, transport.ErrInvalidFrame
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
