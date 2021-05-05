package packet

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPacketType(t *testing.T) {
	var tests = []struct {
		b       byte
		fType   FrameType
		pType   PacketType
		strByte byte
		binByte byte
		str     string
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

	for i, test := range tests {
		typ := ByteToPacketType(test.b, test.fType)

		require.Equal(t, test.pType, typ, fmt.Sprintf(`types not equal by case: %d`, i))

		assert.Equal(t, test.strByte, typ.StringByte(), fmt.Sprintf(`string byte not equal by case: %d`, i))
		assert.Equal(t, test.binByte, typ.BinaryByte(), fmt.Sprintf(`bytes not equal by case: %d`, i))
		assert.Equal(t, test.str, typ.String(), fmt.Sprintf(`strings not equal by case: %d`, i))
	}
}
