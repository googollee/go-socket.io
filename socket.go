package socketio

import (
	"net/http"

	"github.com/googollee/go-engine.io"
)

// Socket is the socket object of socket.io.
type Socket interface {

	// Id returns the session id of socket.
	Id() string

	// Rooms returns the rooms name joined now.
	Rooms() []string

	// Request returns the first http request when established connection.
	Request() *http.Request

	// On registers the function f to handle message.
	On(message string, f interface{}) error

	// Emit emits the message with given args.
	Emit(message string, args ...interface{}) error

	// Join joins the room.
	Join(room string) error

	// Leave leaves the room.
	Leave(room string) error

	// BroadcastTo broadcasts the message to the room with given args.
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
