package socketio

import (
	"code.google.com/p/go.net/websocket"
	"net"
	"net/http"
	"sync"
	"time"
)

func init() {
	DefaultTransports.RegisterTransport(WebSocket)
}

var WebSocket = new(webSocket)

type webSocket struct {
	mutex     sync.Mutex
	session   *Session
	conn      *websocket.Conn
	isConnect bool
	isOpen    bool
	heartBeat time.Duration
}

func (ws *webSocket) Name() string {
	return "websocket"
}

func (ws *webSocket) New(session *Session, heartbeatTimeout int) Transport {
	ret := &webSocket{session: session}
	ret.heartBeat = time.Duration(heartbeatTimeout) * time.Second / 2
	return ret
}

func (ws *webSocket) webSocketHandler(conn *websocket.Conn) {
	ws.mutex.Lock()
	if ws.isOpen {
		ws.Close()
	}
	ws.conn = conn
	ws.isOpen = true
	ws.mutex.Unlock()
	ws.session.onOpen()
	for {
		var data string
		ws.conn.SetDeadline(time.Now().Add(ws.heartBeat))
		err := websocket.Message.Receive(conn, &data)
		if e, ok := err.(net.Error); ok && e.Timeout() {
			heartBeat := new(heartbeatPacket)
			err = ws.session.Of("").sendPacket(heartBeat)
			if err != nil {
				ws.Close()
				return
			}
			continue
		}
		if err != nil {
			ws.Close()
			return
		}
		ws.session.onFrame([]byte(data))
	}
}

func (ws *webSocket) OnData(w http.ResponseWriter, r *http.Request) {
	websocket.Handler(ws.webSocketHandler).ServeHTTP(w, r)
}

func (ws *webSocket) Send(data []byte) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.conn.SetDeadline(time.Now().Add(ws.heartBeat))
	return websocket.Message.Send(ws.conn, string(data))
}

func (ws *webSocket) Close() {
	ws.isOpen = false
	disconnect := new(disconnectPacket)
	for _, ns := range ws.session.nameSpaces {
		if ns.Endpoint() == "" {
			continue
		}
		ns.sendPacket(disconnect)
	}
	ws.session.Of("").sendPacket(disconnect)
	ws.conn.Close()
}
