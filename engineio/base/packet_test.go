package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPacketType(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		b         byte
		frameType FrameType
		typ       PacketType
		strbyte   byte
		binbyte   byte
		str       string
	}{
		{0, FrameBinary, OPEN, '0', 0, "open"},
		{1, FrameBinary, CLOSE, '1', 1, "close"},
		{2, FrameBinary, PING, '2', 2, "ping"},
		{3, FrameBinary, PONG, '3', 3, "pong"},
		{4, FrameBinary, MESSAGE, '4', 4, "message"},
		{5, FrameBinary, UPGRADE, '5', 5, "upgrade"},
		{6, FrameBinary, NOOP, '6', 6, "noop"},

		{'0', FrameString, OPEN, '0', 0, "open"},
		{'1', FrameString, CLOSE, '1', 1, "close"},
		{'2', FrameString, PING, '2', 2, "ping"},
		{'3', FrameString, PONG, '3', 3, "pong"},
		{'4', FrameString, MESSAGE, '4', 4, "message"},
		{'5', FrameString, UPGRADE, '5', 5, "upgrade"},
		{'6', FrameString, NOOP, '6', 6, "noop"},
	}

	for _, test := range tests {
		typ := ByteToPacketType(test.b, test.frameType)
		at.Equal(test.typ, typ)
		at.Equal(test.strbyte, typ.StringByte())
		at.Equal(test.binbyte, typ.BinaryByte())
		at.Equal(test.str, typ.String())
		at.Equal(test.str, PacketType(typ).String())
	}
}
