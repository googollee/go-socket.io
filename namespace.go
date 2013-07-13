package socketio

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"time"
)

type NameSpace struct {
	*EventEmitter
	Name        string
	session     *Session
	id          int
	waitingLock sync.Mutex
	waiting     map[int]chan []byte
	onMessage   func([]byte) interface{}
}

func NewNameSpace(session *Session, name string, ee *EventEmitter) *NameSpace {
	ret := &NameSpace{
		EventEmitter: ee,
		Name:         name,
		session:      session,
		id:           1,
		waiting:      make(map[int]chan []byte),
	}
	if ret.EventEmitter == nil {
		ret.EventEmitter = NewEventEmitter()
	}
	return ret
}

func (ns *NameSpace) Of(name string) *NameSpace {
	return ns.session.Of(name)
}

func (ns *NameSpace) Call(name string, reply []interface{}, args ...interface{}) error {
	var c chan []byte

	pack := new(eventPacket)
	pack.endPoint = ns.Name
	pack.name = name
	if len(reply) > 0 {
		c = make(chan []byte)
		ns.waitingLock.Lock()
		id := ns.id
		ns.id++
		ns.waiting[id] = c
		ns.waitingLock.Unlock()

		pack.id = id
		pack.ack = true
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

	if len(reply) > 0 {
		select {
		case replyRaw := <-c:
			err := json.Unmarshal(replyRaw, reply)
			if err != nil {
				return err
			}
		case <-time.After(time.Duration(ns.session.server.heartbeatTimeout) * time.Second):
			return errors.New("time out")
		}
	}

	return nil
}

func (ns *NameSpace) onAckPacket(packet *ackPacket) {
	c := func() chan []byte {
		ns.waitingLock.Lock()
		defer ns.waitingLock.Unlock()
		if c, ok := ns.waiting[packet.Id()]; ok {
			delete(ns.waiting, packet.Id())
			return c
		}
		return nil
	}()
	if c != nil {
		c <- []byte(packet.args)
	}
}

func (ns *NameSpace) onMessagePacket(packet messageMix) {
	message, ok := packet.(messageMix)
	if !ok {
		return
	}
	data := message.Data()
	result := ns.onMessage(data)
	if message.Id() == 0 {
		return
	}
	if !message.Ack() {
		ack := new(ackPacket)
		ack.ackId = packet.Id()
		ack.args = nil
		ack.endPoint = ns.Name
		ns.sendPacket(ack)
		return
	}
	kindOfResult := reflect.ValueOf(result).Kind()
	var ackData []byte
	if result != nil && kindOfResult != reflect.Invalid {
		if kindOfResult == reflect.Array || kindOfResult == reflect.Slice {
			ackData, _ = json.Marshal(result)
		} else {
			ackData, _ = json.Marshal([]interface{}{result})
		}
	}
	ack := new(ackPacket)
	ack.ackId = message.Id()
	ack.args = ackData
	ack.endPoint = ns.Name
	ns.sendPacket(ack)
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
		ack.endPoint = ns.Name
		ns.sendPacket(ack)
	}
	if packet.Id() == 0 {
		callback = nil
	}
	ns.emitRaw(packet.name, ns, callback, packet.args)
}

func (ns *NameSpace) sendPacket(packet Packet) error {
	return ns.session.transport.Send(encodePacket(ns.Name, packet))
}

func (ns *NameSpace) onConnect() {
	ns.emit("connect", ns, nil)
}

func (ns *NameSpace) onDisconnect() {
	ns.emit("disconnect", ns, nil)
}
