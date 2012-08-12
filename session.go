package socketio

import (
	"crypto/rand"
	"io"
	"net/http"
	"sync"
)

const (
	SessionIDLength  = 16
	SessionIDCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

func NewSessionID() string {
	b := make([]byte, SessionIDLength)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}

	for i := 0; i < SessionIDLength; i++ {
		b[i] = SessionIDCharset[b[i]%uint8(len(SessionIDCharset))]
	}

	return string(b)
}

type Session struct {
	mutex        sync.Mutex
	server       *SocketIOServer
	SessionId    string
	nameSpaces   map[string]*NameSpace
	transport    Transport
	onConnect    func(*NameSpace)
	onDisconnect func(*NameSpace)
}

func NewSession(server *SocketIOServer, sessionId string) *Session {
	return &Session{server: server, SessionId: sessionId, nameSpaces: make(map[string]*NameSpace)}
}

func (ss *Session) Of(name string) (nameSpace *NameSpace) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if nameSpace, ok := ss.nameSpaces[name]; !ok {
		nameSpace = NewNameSpace(ss, name)
		ss.nameSpaces[name] = nameSpace
	}
	return
}

func (ss *Session) serve(transportId string, w http.ResponseWriter, r *http.Request) {
	if ss.transport == nil {
		ss.transport = ss.server.transports.Get(transportId).New(ss)
		ss.transport.OnOpen(w, r)
		return
	}
	ss.transport.OnData(w, r)
}

func (ss *Session) onPacket(packet Packet) {

}

func (ss *Session) onError(err error) {

}

func (ss *Session) connected(ns *NameSpace) {
	ss.onConnect(ns)
	ss.server.onConnect(ns)
}

func (ss *Session) disconnected(ns *NameSpace) {
	ss.onDisconnect(ns)
	ss.server.onDisconnect(ns)
}

func (ss *Session) OnConnect(fn func(*NameSpace)) {
	ss.onConnect = fn
}

func (ss *Session) OnDisconnect(fn func(*NameSpace)) {
	ss.onDisconnect = fn
}
