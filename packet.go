package socketio

import (
	"encoding/json"
)

const (
	PACKET_DISCONNECT = iota
	PACKET_CONNECT
	PACKET_HEARTBEAT
	PACKET_MESSAGE
	PACKET_JSONMESSAGE
	PACKET_EVENT
	PACKET_ACK
	PACKET_ERROR
	PACKET_NOOP
)

type MessageType uint8

type Packet interface {
	Id() int
	Type() MessageType
	EndPoint() string
	Ack() bool
}

type packetCommon struct {
	id       int
	endPoint string
	ack      bool
}

func (p *packetCommon) Id() int {
	return p.id
}

func (p *packetCommon) EndPoint() string {
	return p.endPoint
}

func (p *packetCommon) Ack() bool {
	return p.ack
}

type disconnectPacket struct {
	packetCommon
}

func (*disconnectPacket) Type() MessageType {
	return PACKET_DISCONNECT
}

type connectPacket struct {
	packetCommon
	query string
}

func (*connectPacket) Type() MessageType {
	return PACKET_CONNECT
}

type heartbeatPacket struct {
	packetCommon
}

func (*heartbeatPacket) Type() MessageType {
	return PACKET_HEARTBEAT
}

type messageMix interface {
	Packet
	Data() []byte
}

type messagePacket struct {
	packetCommon
	data []byte
}

func (*messagePacket) Type() MessageType {
	return PACKET_MESSAGE
}

func (p *messagePacket) Data() []byte {
	return p.data
}

type jsonPacket struct {
	packetCommon
	data []byte
}

func (*jsonPacket) Type() MessageType {
	return PACKET_JSONMESSAGE
}

func (p *jsonPacket) Data() []byte {
	return p.data
}

type eventPacket struct {
	packetCommon
	name string
	args json.RawMessage
}

func (*eventPacket) Type() MessageType {
	return PACKET_EVENT
}

type ackPacket struct {
	packetCommon
	ackId int
	args  json.RawMessage
}

func (*ackPacket) Type() MessageType {
	return PACKET_ACK
}

type errorPacket struct {
	packetCommon
	reason string
	advice string
}

func (*errorPacket) Type() MessageType {
	return PACKET_ERROR
}
