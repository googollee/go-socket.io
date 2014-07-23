package socketio

import (
	"net/http"

	"github.com/googollee/go-engine.io"
)

type Socket interface {
	Id() string
	Rooms() []string
	Request() *http.Request
	On(message string, f interface{}) error
	Emit(message string, args ...interface{}) error
	Join(room string) error
	Leave(room string) error
	BroadcastTo(room, message string, args ...interface{}) error
}

type socket struct {
	*socketHandler
	conn      engineio.Conn
	namespace string
	id        int
}

func newSocket(conn engineio.Conn, base *baseHandler) *socket {
	ret := &socket{
		conn: conn,
	}
	ret.socketHandler = newSocketHandler(ret, base)
	return ret
}

func (s *socket) Id() string {
	return s.conn.Id()
}

func (s *socket) Request() *http.Request {
	return s.conn.Request()
}

func (s *socket) Emit(message string, args ...interface{}) error {
	if err := s.socketHandler.Emit(message, args...); err != nil {
		return err
	}
	if message == "disconnect" {
		s.conn.Close()
	}
	return nil
}

func (s *socket) send(args []interface{}) error {
	packet := Packet{
		Type: EVENT,
		Id:   -1,
		NSP:  s.namespace,
		Data: args,
	}
	encoder := NewEncoder(s.conn)
	return encoder.Encode(packet)
}

func (s *socket) sendId(args []interface{}) (int, error) {
	packet := Packet{
		Type: EVENT,
		Id:   s.id,
		NSP:  s.namespace,
		Data: args,
	}
	s.id++
	if s.id < 0 {
		s.id = 0
	}
	encoder := NewEncoder(s.conn)
	err := encoder.Encode(packet)
	if err != nil {
		return -1, nil
	}
	return packet.Id, nil
}

func (s *socket) loop() error {
	defer func() {
		s.LeaveAll()
		packet := Packet{
			Type: DISCONNECT,
			Id:   -1,
		}
		s.socketHandler.onPacket(nil, &packet)
	}()

	packet := Packet{
		Type: CONNECT,
		Id:   -1,
	}
	encoder := NewEncoder(s.conn)
	if err := encoder.Encode(packet); err != nil {
		return err
	}
	s.socketHandler.onPacket(nil, &packet)
	for {
		decoder := NewDecoder(s.conn)
		var packet Packet
		if err := decoder.Decode(&packet); err != nil {
			return err
		}
		ret, err := s.socketHandler.onPacket(decoder, &packet)
		if err != nil {
			return err
		}
		switch packet.Type {
		case CONNECT:
			s.namespace = packet.NSP
		case BINARY_EVENT:
			fallthrough
		case EVENT:
			if packet.Id >= 0 {
				packet := Packet{
					Type: ACK,
					Id:   packet.Id,
					NSP:  s.namespace,
					Data: ret,
				}
				encoder := NewEncoder(s.conn)
				if err := encoder.Encode(packet); err != nil {
					return err
				}
			}
		case DISCONNECT:
			return nil
		}
	}
}
