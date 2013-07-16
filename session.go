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
	SessionId         string
	mutex             sync.Mutex
	emitters          map[string]*EventEmitter
	nameSpaces        map[string]*NameSpace
	transport         Transport
	heartbeatTimeout  time.Duration
	connectionTimeout time.Duration
	peerLast          time.Time
	isConnected       bool
	sendHeartBeat     bool
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

func NewSession(emitters map[string]*EventEmitter, sessionId string, timeout int, sendHeartbeat bool) *Session {
	ret := &Session{
		emitters:      emitters,
		SessionId:     sessionId,
		nameSpaces:    make(map[string]*NameSpace),
		sendHeartBeat: sendHeartbeat,
	}
	ret.heartbeatTimeout = time.Duration(timeout) * time.Second * 2 / 3
	ret.connectionTimeout = time.Duration(timeout) * time.Second
	return ret
}

func (ss *Session) Of(name string) (nameSpace *NameSpace) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if nameSpace = ss.nameSpaces[name]; nameSpace == nil {
		ee := ss.emitters[name]
		if ee == nil {
			return nil
		}
		nameSpace = NewNameSpace(ss, name, ee)
		ss.nameSpaces[name] = nameSpace
	}
	return
}

func (ss *Session) loop() {
	if ss.sendHeartBeat {
		ss.onOpen()
	}
	ss.peerLast = time.Now()
	last := time.Now()
	for {
		now := time.Now()
		if ss.sendHeartBeat && now.Sub(last) > ss.heartbeatTimeout {
			last = now
			if err := ss.heartbeat(); err != nil {
				ss.isConnected = false
			}
		}
		if now.Sub(ss.peerLast) > ss.connectionTimeout {
			ss.isConnected = false
			return
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
	connected := ss.isConnected
	ss.isConnected = true
	err := ss.Of("").sendPacket(new(heartbeatPacket))
	ss.isConnected = connected
	return err
}

func (ss *Session) onFrame(data []byte) {
	packet, err := decodePacket(data)
	if err != nil || packet == nil {
		return
	}
	ss.onPacket(packet)
}

func (ss *Session) onPacket(packet Packet) {
	ns := ss.Of(packet.EndPoint())
	if ns == nil {
		return
	}
	switch p := packet.(type) {
	case *heartbeatPacket:
		ss.peerLast = time.Now()
		ss.isConnected = true
		if !ss.sendHeartBeat {
			ss.Of(p.endPoint).sendPacket(new(heartbeatPacket))
		}
	case *disconnectPacket:
		ns.onDisconnect()
	case *connectPacket:
		ss.isConnected = true
		ns.onConnect()
	case *messagePacket, *jsonPacket:
		ns.onMessagePacket(p.(messageMix))
	case *eventPacket:
		ns.onEventPacket(p)
	case *ackPacket:
		ns.onAckPacket(p)
	}
}

func (ss *Session) onOpen() {
	packet := new(connectPacket)
	ss.isConnected = true
	ns := ss.Of("")
	err := ns.sendPacket(packet)
	if err == nil {
		ns.onConnect()
	} else {
		ss.isConnected = false
	}
}
