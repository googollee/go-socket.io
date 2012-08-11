package socketio

import (
	"encoding/json"
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
	return 1
}

type messagePacket struct {
	*packetCommon
	data []byte
}

func (*messagePacket) Type() MessageType {
	return 3
}

type jsonPacket struct {
	*packetCommon
	data []byte
}

func (*jsonPacket) Type() MessageType {
	return 4
}

type eventPacket struct {
	*packetCommon
	name string
	args argList
}

func (*eventPacket) Type() MessageType {
	return 5
}

type ackPacket struct {
	*packetCommon
	ackId int
	args  argList
}

func (*ackPacket) Type() MessageType {
	return 6
}

type errorPacket struct {
	*packetCommon
	reason string
	advice string
}

func (*errorPacket) Type() MessageType {
	return 7
}
