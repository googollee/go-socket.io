package session

import (
	"github.com/googollee/go-socket.io/engineio/packet"
)

// FrameType is type of a message frame.
type FrameType packet.FrameType

const (
	// TEXT is text type message.
	TEXT = FrameType(packet.FrameString)
	// BINARY is binary type message.
	BINARY = FrameType(packet.FrameBinary)
)
