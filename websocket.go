package engineio

import (
	"io"
	"net/http"

	ws "github.com/gorilla/websocket"
)

func init() {
	registerTransport("websocket", true, newWebsocketTransport)
}

type websocket struct {
	socket   Conn
	conn     *ws.Conn
	isClosed bool
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
	p.conn = conn
	defer func() {
		p.socket.onClose()
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
	w, err := p.conn.NextWriter(wsType)
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
	if w, _ := p.conn.NextWriter(ws.CloseMessage); w != nil {
		w.Close()
	}
	return p.conn.Close()
}
