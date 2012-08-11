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

type argList []json.RawMessage

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

type connectPacket struct {
	*packetCommon
	query string
}

func (*connectPacket) Type() MessageType {
	return PACKET_CONNECT
}

type messagePacket struct {
	*packetCommon
	data []byte
}

func (*messagePacket) Type() MessageType {
	return PACKET_MESSAGE
}

type jsonPacket struct {
	*packetCommon
	data []byte
}

func (*jsonPacket) Type() MessageType {
	return PACKET_JSONMESSAGE
}

type eventPacket struct {
	*packetCommon
	name string
	args argList
}

func (*eventPacket) Type() MessageType {
	return PACKET_EVENT
}

type ackPacket struct {
	*packetCommon
	ackId int
	args  argList
}

func (*ackPacket) Type() MessageType {
	return PACKET_ACK
}

type errorPacket struct {
	*packetCommon
	reason string
	advice string
}

func (*errorPacket) Type() MessageType {
	return PACKET_ERROR
}
