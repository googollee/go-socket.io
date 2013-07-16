package socketio

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"io"
	"time"
)

func init() {
	DefaultTransports.RegisterTransport("websocket")
}

type webSocket struct {
	session *Session
	conn    *websocket.Conn
}

func newWebSocket(session *Session, timeout int) *webSocket {
	ret := &webSocket{
		session: session,
	}
	session.transport = ret
	return ret
}

func (ws *webSocket) Send(data []byte) error {
	ws.conn.SetWriteDeadline(time.Now().Add(ws.session.heartbeatTimeout))
	return websocket.Message.Send(ws.conn, string(data))
}

func (ws *webSocket) Read() (io.Reader, error) {
	var ret string
	ws.conn.SetReadDeadline(time.Now().Add(ws.session.heartbeatTimeout))
	err := websocket.Message.Receive(ws.conn, &ret)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewBufferString(ret)
	return reader, nil
}

func (ws *webSocket) webSocketHandler(conn *websocket.Conn) {
	ws.conn = conn
	ws.session.loop()
}
