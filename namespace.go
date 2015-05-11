package socketio

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type NameSpace struct {
	*EventEmitter
	endpoint    string
	Session     *Session
	connected   bool
	id          int
	waitingLock sync.Mutex
	waiting     map[int]chan []byte
}

func NewNameSpace(session *Session, endpoint string, ee *EventEmitter) *NameSpace {
	ret := &NameSpace{
		EventEmitter: ee,
		endpoint:     endpoint,
		Session:      session,
		connected:    false,
		id:           1,
		waiting:      make(map[int]chan []byte),
	}
	return ret
}

func (ns *NameSpace) Endpoint() string {
	return ns.endpoint
}

func (ns *NameSpace) Id() string {
	return ns.Session.SessionId
}

func (ns *NameSpace) Call(name string, timeout time.Duration, reply []interface{}, args ...interface{}) error {
	if !ns.connected {
		return NotConnected
	}

	var c chan []byte
	pack := new(eventPacket)
	pack.endPoint = ns.endpoint
	pack.name = name
	if len(reply) > 0 {
		pack.ack = true
		c = make(chan []byte)

		ns.waitingLock.Lock()
		pack.id = ns.id
		ns.id++
		ns.waiting[pack.id] = c
		ns.waitingLock.Unlock()

		defer func() {
			ns.waitingLock.Lock()
			defer ns.waitingLock.Unlock()
			delete(ns.waiting, pack.id)
		}()
	}

	var err error
	pack.args, err = json.Marshal(args)
	if err != nil {
		return err
	}
	err = ns.sendPacket(pack)
	if err != nil {
		return err
	}

	if c != nil {
		select {
		case replyRaw := <-c:
			err := json.Unmarshal(replyRaw, &reply)
			if err != nil {
				return err
			}
		case <-time.After(timeout):
			return errors.New("time out")
		}
	}

	return nil
}

func (ns *NameSpace) Emit(name string, args ...interface{}) error {
	if !ns.connected {
		return NotConnected
	}

	pack := new(eventPacket)
	pack.endPoint = ns.endpoint
	pack.name = name

	var err error
	switch len(args) {
	case 0: // marshal as empty object
		pack.args, err = json.Marshal(map[string]string{})
	case 1: // marshal as single object
		pack.args, err = json.Marshal(args[0])
	default: // marshal as array of objects
		pack.args, err = json.Marshal(args)
	}

	err = ns.sendPacket(pack)
	if err != nil {
		return err
	}
	return nil
}

func (ns *NameSpace) onPacket(packet Packet) {
	switch p := packet.(type) {
	case *disconnectPacket:
		ns.onDisconnect()
	case *connectPacket:
		ns.onConnect()
	case *eventPacket:
		ns.onEventPacket(p)
	case *ackPacket:
		ns.onAckPacket(p)
	}
}

func (ns *NameSpace) onAckPacket(packet *ackPacket) {
	c, ok := ns.waiting[packet.ackId]
	if !ok {
		return
	}
	c <- []byte(packet.args)
}

func (ns *NameSpace) onEventPacket(packet *eventPacket) {
	callback := func(args []interface{}) {
		ack := new(ackPacket)
		ack.ackId = packet.Id()
		ackData, err := json.Marshal(args)
		if err != nil {
			return
		}
		ack.args = ackData
		ack.endPoint = ns.endpoint
		ns.sendPacket(ack)
	}
	if packet.Id() == 0 {
		callback = nil
	}
	ns.emitRaw(packet.name, ns, callback, packet.args, packet.packetCommon)
}

func (ns *NameSpace) sendPacket(packet Packet) error {
	if !ns.connected {
		println(ns.endpoint + "not connected")
		return NotConnected
	}
	return ns.Session.transport.Send(encodePacket(ns.endpoint, packet))
}

func (ns *NameSpace) onConnect() {
	if ns.connected == false {
		ns.emit("connect", ns, nil)
		ns.connected = true
		ns.Emit("connect")
	}
}

func (ns *NameSpace) onDisconnect() {
	ns.emit("disconnect", ns, nil)
	ns.connected = false
}
