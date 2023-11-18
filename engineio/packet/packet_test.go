package packet

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/googollee/go-socket.io/engineio/frame"
)

func TestPacketType(t *testing.T) {
	var tests = []struct {
		b       byte
		fType   frame.Type
		pType   Type
		strByte byte
		binByte byte
		str     string
	}{
		{0, frame.Binary, OPEN, '0', 0, "open"},
		{1, frame.Binary, CLOSE, '1', 1, "close"},
		{2, frame.Binary, PING, '2', 2, "ping"},
		{3, frame.Binary, PONG, '3', 3, "pong"},
		{4, frame.Binary, MESSAGE, '4', 4, "message"},
		{5, frame.Binary, UPGRADE, '5', 5, "upgrade"},
		{6, frame.Binary, NOOP, '6', 6, "noop"},

		{'0', frame.String, OPEN, '0', 0, "open"},
		{'1', frame.String, CLOSE, '1', 1, "close"},
		{'2', frame.String, PING, '2', 2, "ping"},
		{'3', frame.String, PONG, '3', 3, "pong"},
		{'4', frame.String, MESSAGE, '4', 4, "message"},
		{'5', frame.String, UPGRADE, '5', 5, "upgrade"},
		{'6', frame.String, NOOP, '6', 6, "noop"},
	}

	for i, test := range tests {
		typ := ByteToPacketType(test.b, test.fType)

		require.Equal(t, test.pType, typ, fmt.Sprintf(`types not equal by case: %d`, i))

		assert.Equal(t, test.strByte, typ.StringByte(), fmt.Sprintf(`string byte not equal by case: %d`, i))
		assert.Equal(t, test.binByte, typ.BinaryByte(), fmt.Sprintf(`bytes not equal by case: %d`, i))
		assert.Equal(t, test.str, typ.String(), fmt.Sprintf(`strings not equal by case: %d`, i))
	}
}
