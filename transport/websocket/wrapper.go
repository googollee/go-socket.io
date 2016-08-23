package websocket

import (
	"io"
	"io/ioutil"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/transport"
	"github.com/gorilla/websocket"
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
	for {
		typ, r, err := w.Conn.NextReader()
		if err != nil {
			return 0, nil, err
		}
		ret, ok := r.(io.ReadCloser)
		if !ok {
			ret = ioutil.NopCloser(r)
		}
		switch typ {
		case websocket.TextMessage:
			return base.FrameString, ret, nil
		case websocket.BinaryMessage:
			return base.FrameBinary, ret, nil
		}
	}
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
