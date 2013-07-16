package socketio

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
)

var (
	packetSep    = []byte{0xff, 0xfd}
	packetRegexp = regexp.MustCompile(`^([^:]+):([0-9]+)?(\+)?:([^:]+)?:?(.*)?$`)
)

type Event struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

func encodePacket(endpoint string, packet Packet) []byte {
	buf := &bytes.Buffer{}
	buf.WriteString(strconv.Itoa(int(packet.Type())))
	buf.WriteByte(':')
	if packet.Id() != 0 {
		buf.WriteString(strconv.Itoa(packet.Id()))
		if packet.Ack() {
			buf.WriteByte('+')
		}
	}
	buf.WriteByte(':')
	buf.WriteString(endpoint)
	buf.WriteByte(':')

	enc := json.NewEncoder(buf)
	switch p := packet.(type) {
	case *connectPacket:
		buf.WriteString(p.query)
	case *ackPacket:
		ackIdStr := strconv.Itoa(p.ackId)
		buf.WriteString(ackIdStr)
		if p.args != nil {
			buf.WriteByte('+')
			buf.Write(p.args)
		}
	case *jsonPacket:
		buf.Write(p.data)
	case *messagePacket:
		buf.Write(p.data)
	case *eventPacket:
		event := &Event{Name: p.name, Args: p.args}
		err := enc.Encode(event)
		if err != nil {
			return nil
		}
	case *errorPacket:
		buf.WriteString(p.reason)
		if p.advice != "" {
			buf.WriteByte('+')
			buf.WriteString(p.advice)
		}
	}

	return buf.Bytes()
}

func encodePayload(payloads [][]byte) []byte {
	if len(payloads) == 1 {
		return payloads[0]
	}
	buf := &bytes.Buffer{}
	for _, payload := range payloads {
		buf.Write(packetSep)
		buf.WriteString(strconv.Itoa(len(payload)))
		buf.Write(packetSep)
		buf.Write(payload)
	}
	return buf.Bytes()
}

func decodePacket(b []byte) (packet Packet, err error) {
	b = bytes.Trim(b, "\n \r\t")
	pieces := packetRegexp.FindSubmatch(b)
	if pieces == nil {
		return nil, errors.New("invalid packet")
	}
	var tid int
	tid, err = strconv.Atoi(string(pieces[1]))
	if err != nil {
		return
	}
	common := packetCommon{}
	if len(pieces[2]) == 0 {
		common.id = -1
	} else {
		common.id, err = strconv.Atoi(string(pieces[2]))
		if err != nil {
			return
		}
	}
	common.ack = string(pieces[3]) == "+"
	common.endPoint = string(pieces[4])
	data := pieces[5]
	switch tid {
	case 0: // disconnect
		p := new(disconnectPacket)
		p.packetCommon = common
		packet = p
	case 1: // connect
		p := new(connectPacket)
		p.packetCommon = common
		p.query = string(data)
		packet = p
	case 2: // heartbeat
		p := new(heartbeatPacket)
		p.packetCommon = common
		packet = p
	case 3: // message
		p := new(messagePacket)
		p.packetCommon = common
		p.data = data
		packet = p
	case 4: //jsonmessage
		p := new(jsonPacket)
		p.packetCommon = common
		p.data = data
		packet = p
	case 5: //event
		event := new(Event)
		err = json.Unmarshal(data, event)
		if err != nil {
			return
		}
		p := new(eventPacket)
		p.packetCommon = common
		p.name = event.Name
		p.args = event.Args
		packet = p
	case 6: // ack
		p := new(ackPacket)
		p.packetCommon = common
		pos := bytes.Index(data, []byte{'+'})
		var ackId int
		var args json.RawMessage
		if pos < 0 {
			ackId, err = strconv.Atoi(string(data))
			if err != nil {
				return
			}
		} else {
			ackId, err = strconv.Atoi(string(data[0:pos]))
			if err != nil {
				return
			}
			err = json.Unmarshal(data[pos+1:], &args)
			if err != nil {
				return
			}
		}
		p.ackId = ackId
		p.args = args
		packet = p
	case 7: //error
		p := new(errorPacket)
		p.packetCommon = common
		pos := bytes.Index(data, []byte{'+'})
		if pos < 0 {
			p.reason = string(data)
		} else {
			p.reason = string(data[0:pos])
			p.advice = string(data[pos+1:])
		}
		packet = p
	default:
		return nil, errors.New("invalid message type")
	}
	return
}

func decodePayload(data []byte) (packets []Packet, err error) {
	if len(data) >= 2 && bytes.Equal(data[0:2], packetSep) {
		for {
			if len(data) == 0 {
				break
			}
			data = data[2:]
			pos := bytes.Index(data[2:], packetSep)
			var length int
			length, err = strconv.Atoi(string(data[0:pos]))
			if err != nil {
				return
			}
			data = data[pos+2:]
			var packet Packet
			packet, err = decodePacket(data[0:length])
			if err != nil {
				return
			}
			packets = append(packets, packet)
			data = data[length:]
		}
	} else {
		var packet Packet
		packet, err = decodePacket(data)
		if err != nil {
			return
		}
		return []Packet{packet}, nil
	}
	panic("not reached")
	return
}
