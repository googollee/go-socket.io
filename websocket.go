package socketio

import (
	"code.google.com/p/go.net/websocket"
	"net/http"
)

func init() {
	DefaultTransports.RegisterTransport(WebSocket)
}

var WebSocket = new(webSocket)

type webSocket struct {
	session *Session
}

func (ws *webSocket) Name() string {
	return "websocket"
}

func (ws *webSocket) New(session *Session) Transport {
	return &webSocket{session: session}

}

func (ws *webSocket) OnOpen(w http.ResponseWriter, r *http.Request) {
	webSocketHandler := func(conn *websocket.Conn) {
		ws.webSocketHandler(conn)
	}
	go websocket.Handler(webSocketHandler).ServeHTTP(w, r)
}

func (ws *webSocket) webSocketHandler(conn *websocket.Conn) {
	for {
		var data []byte
		websocket.Message.Receive(conn, &data)
		packet, err := decodePacket(data)
		if err != nil {
			ws.session.onError(err)
			break
		}
		ws.session.onPacket(packet)

	}
}

func (ws *webSocket) OnData(w http.ResponseWriter, r *http.Request) {

}
