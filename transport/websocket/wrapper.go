package websocket

import (
	"io"
	"io/ioutil"

	"github.com/gorilla/websocket"
	"gopkg.in/googollee/go-engine.io.v1/base"
	"gopkg.in/googollee/go-engine.io.v1/transport"
)

type wrapper struct {
	*websocket.Conn
}

func newWrapper(conn *websocket.Conn) wrapper {
	return wrapper{
		Conn: conn,
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
	return w.Conn.NextWriter(t)
}
