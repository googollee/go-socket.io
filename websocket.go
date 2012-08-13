package socketio

import (
	"code.google.com/p/go.net/websocket"
	"net/http"
	"sync"
)

func init() {
	DefaultTransports.RegisterTransport(WebSocket)
}

var WebSocket = new(webSocket)

type webSocket struct {
	mutex       sync.Mutex
	session     *Session
	conn        *websocket.Conn
	isConnect   bool
	isOpen      bool
	waitForOpen chan bool
}

func (ws *webSocket) Name() string {
	return "websocket"
}

func (ws *webSocket) New(session *Session) Transport {
	return &webSocket{session: session}
}

func (ws *webSocket) webSocketHandler(conn *websocket.Conn) {
	ws.mutex.Lock()
	if ws.isOpen {
		ws.Close()
	}
	ws.conn = conn
	ws.isOpen = true
	ws.mutex.Unlock()
	for {
		var data []byte
		err := websocket.Message.Receive(conn, &data)
		if err != nil {
			ws.Close()
		}
		ws.session.onFrame(data)
	}
}

func (ws *webSocket) OnData(w http.ResponseWriter, r *http.Request) {
	webSocketHandler := func(conn *websocket.Conn) {
		ws.webSocketHandler(conn)
	}
	go websocket.Handler(webSocketHandler).ServeHTTP(w, r)
}

func (ws *webSocket) Send(data []byte) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	websocket.Message.Send(ws.conn, data)
}

func (ws *webSocket) Close() {
	ws.isOpen = false
	ws.conn.Close()
}
