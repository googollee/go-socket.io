package payload

import (
	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/packet"
)

type Packet struct {
	ft   frame.Type
	pt   packet.Type
	data []byte
}

var tests = []struct {
	supportBinary bool
	data          []byte
	packets       []Packet
}{
	{true, []byte{0x00, 0x01, 0xff, '0'}, []Packet{
		{frame.String, packet.OPEN, []byte{}},
	},
	},
	{true, []byte{0x00, 0x01, 0x03, 0xff, '4', 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd},
		[]Packet{
			Packet{frame.String, packet.MESSAGE, []byte("hello 你好")},
		},
	},
	{true, []byte{0x01, 0x01, 0x03, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd}, []Packet{
		Packet{frame.Binary, packet.MESSAGE, []byte("hello 你好")},
	},
	},
	{true, []byte{
		0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
		0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
		0x00, 0x06, 0xff, '2', 'p', 'r', 'o', 'b', 'e',
	}, []Packet{
		Packet{frame.Binary, packet.MESSAGE, []byte("hello\n")},
		Packet{frame.String, packet.MESSAGE, []byte("你好\n")},
		Packet{frame.String, packet.PING, []byte("probe")},
	},
	},
	{false, []byte("1:0"), []Packet{
		Packet{frame.String, packet.OPEN, []byte{}},
	},
	},
	{false, []byte("13:4hello 你好"), []Packet{
		Packet{frame.String, packet.MESSAGE, []byte("hello 你好")},
	},
	},
	{false, []byte("18:b4aGVsbG8g5L2g5aW9"), []Packet{
		Packet{frame.Binary, packet.MESSAGE, []byte("hello 你好")},
	},
	},
	{false, []byte("10:b4aGVsbG8K8:4你好\n6:2probe"), []Packet{
		Packet{frame.Binary, packet.MESSAGE, []byte("hello\n")},
		Packet{frame.String, packet.MESSAGE, []byte("你好\n")},
		Packet{frame.String, packet.PING, []byte("probe")},
	},
	},
}
