package engineio

import (
	"io"
	"net/http"
	"sync"

	ws "github.com/gorilla/websocket"
)

func init() {
	registerTransport("websocket", true, newWebsocketTransport)
}

type websocket struct {
	socket     Conn
	conn       *ws.Conn
	connLocker sync.RWMutex
	isClosed   bool
}

func newWebsocketTransport(req *http.Request) (transport, error) {
	ret := &websocket{
		isClosed: false,
	}
	return ret, nil
}

func (*websocket) Name() string {
	return "websocket"
}

func (p *websocket) SetConn(s Conn) {
	p.socket = s
}

func (*websocket) SupportsFraming() bool {
	return true
}

func (p *websocket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.Upgrade(w, r, nil, 10240, 10240)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	p.connLocker.Lock()
	p.conn = conn
	p.connLocker.Unlock()
	defer func() {
		p.connLocker.Lock()
		defer p.connLocker.Unlock()
		if p.conn != nil {
			p.socket.onClose()
		}
	}()

	for {
		t, r, err := conn.NextReader()
		if err != nil {
			return
		}

		if t == ws.TextMessage || t == ws.BinaryMessage {
			decoder, err := newDecoder(r)
			if err != nil {
				return
			}
			p.socket.onPacket(decoder)
			decoder.Close()
		}
	}
}

func (p *websocket) NextWriter(msgType MessageType, packetType packetType) (io.WriteCloser, error) {
	wsType, newEncoder := ws.TextMessage, newStringEncoder
	if msgType == MessageBinary {
		wsType, newEncoder = ws.BinaryMessage, newBinaryEncoder
	}

	w, err := func() (io.WriteCloser, error) {
		p.connLocker.RLock()
		defer p.connLocker.RUnlock()

		if p.conn == nil {
			return nil, io.EOF
		}

		return p.conn.NextWriter(wsType)
	}()

	if err != nil {
		return nil, err
	}
	ret, err := newEncoder(w, packetType)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (p *websocket) Close() error {
	conn := func() *ws.Conn {
		p.connLocker.RLock()
		defer p.connLocker.RUnlock()

		if p.conn == nil {
			return nil
		}

		if w, _ := p.conn.NextWriter(ws.CloseMessage); w != nil {
			w.Close()
		}
		return p.conn
	}()

	if conn == nil {
		return nil
	}
	p.connLocker.Lock()
	p.conn = nil
	p.connLocker.Unlock()
	return conn.Close()
}
