package packet

import "github.com/googollee/go-socket.io/engineio/frame"

// Type is the type of packet
type Type int

const (
	// OPEN is sent from the server when a new transport is opened (recheck).
	OPEN Type = iota
	// CLOSE is request the close of this transport but does not shutdown the
	// connection itself.
	CLOSE
	// PING is sent by the client. Server should answer with a pong packet
	// containing the same data.
	PING
	// PONG is sent by the server to respond to ping packets.
	PONG
	// MESSAGE is actual message, client and server should call their callbacks
	// with the data.
	MESSAGE
	// UPGRADE is sent before engine.io switches a transport to test if server
	// and client can communicate over this transport. If this test succeed,
	// the client sends an upgrade packets which requests the server to flush
	// its cache on the old transport and switch to the new transport.
	UPGRADE
	// NOOP is a noop packet. Used primarily to force a poll cycle when an
	// incoming websocket connection is received.
	NOOP
)

func (id Type) String() string {
	switch id {
	case OPEN:
		return "open"
	case CLOSE:
		return "close"
	case PING:
		return "ping"
	case PONG:
		return "pong"
	case MESSAGE:
		return "message"
	case UPGRADE:
		return "upgrade"
	case NOOP:
		return "noop"
	}
	return "unknown"
}

// StringByte converts a PacketType to byte in string.
func (id Type) StringByte() byte {
	return byte(id) + '0'
}

// BinaryByte converts a PacketType to byte in binary.
func (id Type) BinaryByte() byte {
	return byte(id)
}

// ByteToPacketType converts a byte to PacketType.
func ByteToPacketType(b byte, typ frame.Type) Type {
	if typ == frame.String {
		b -= '0'
	}
	return Type(b)
}
