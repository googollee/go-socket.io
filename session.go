package socketio

import (
	"crypto/rand"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"
)

const (
	SessionIDLength  = 16
	SessionIDCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type Session struct {
	SessionId  string
	mutex      sync.Mutex
	emitters   map[string]*EventEmitter
	nameSpaces map[string]*NameSpace
	transport  Transport
	timeout    time.Duration
}

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

func NewSession(emitters map[string]*EventEmitter, sessionId string, timeout int) *Session {
	return &Session{
		emitters:   emitters,
		SessionId:  sessionId,
		nameSpaces: make(map[string]*NameSpace),
		timeout:    time.Duration(timeout) * time.Second * 2 / 3,
	}
}

func (ss *Session) Of(name string) (nameSpace *NameSpace) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if nameSpace = ss.nameSpaces[name]; nameSpace == nil {
		ee := ss.emitters[name]
		if ee == nil {
			ee = NewEventEmitter()
		}
		nameSpace = NewNameSpace(ss, name, ee)
		ss.nameSpaces[name] = nameSpace
	}
	return
}

func (ss *Session) loop() {
	ss.onOpen()
	last := time.Now()
	for {
		if time.Now().Sub(last) > ss.timeout {
			if err := ss.heartbeat(); err != nil {
				return
			}
			last = time.Now()
		}
		reader, err := ss.transport.Read()
		if e, ok := err.(net.Error); ok && e.Timeout() {
			continue
		}
		if err != nil {
			return
		}
		b, err := ioutil.ReadAll(reader)
		if err != nil {
			return
		}
		ss.onFrame(b)
	}
}

func (ss *Session) heartbeat() error {
	return ss.Of("").sendPacket(new(heartbeatPacket))
}

func (ss *Session) onFrame(data []byte) {
	packet, err := decodePacket(data)
	if err != nil {
		return
	}
	ss.onPacket(packet)
}

func (ss *Session) onPacket(packet Packet) {
	switch p := packet.(type) {
	case *disconnectPacket:
	case *connectPacket:
		ss.Of(packet.EndPoint()).onConnect()
	case *messagePacket, *jsonPacket:
		ss.Of(packet.EndPoint()).onMessagePacket(p.(messageMix))
	case *eventPacket:
		ss.Of(packet.EndPoint()).onEventPacket(p)
	}
}

func (ss *Session) onOpen() {
	packet := new(connectPacket)
	ns := ss.Of("")
	ns.sendPacket(packet)
	ns.onConnect()
}
