package socketio

import (
	"encoding/json"
	"reflect"
)

type NameSpace struct {
	*EventEmitter
	Name       string
	session    *Session
	onMessage  func([]byte) interface{}
	ackPackets int
	acks       map[int]func([]byte)
}

func NewNameSpace(session *Session, name string) *NameSpace {
	return &NameSpace{session: session, Name: name, EventEmitter: NewEventEmitter()}
}

func (ns *NameSpace) Of(name string) *NameSpace {
	return ns.session.Of(name)
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
	if !packet.Ack() {
		callback = nil
		ack := new(ackPacket)
		ack.ackId = packet.Id()
		ack.args = nil
		ack.endPoint = ns.Name
		ns.sendPacket(ack)
	}
	ns.emitRaw(packet.name, callback, packet.args)
}

func (ns *NameSpace) sendPacket(packet Packet) {
	ns.session.transport.Send(encodePacket(packet))
}
