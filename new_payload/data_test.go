package payload

import "github.com/googollee/go-engine.io/base"

type Packet struct {
	ft   base.FrameType
	pt   base.PacketType
	data []byte
}

var tests = []struct {
	supportBinary bool
	packet        Packet
	data          []byte
}{
	{true, Packet{base.FrameString, base.OPEN, []byte{}},
		[]byte{0x00, 0x01, 0xff, '0'},
	},
	{true, Packet{base.FrameString, base.MESSAGE, []byte("hello 你好")},
		[]byte{0x00, 0x01, 0x03, 0xff, '4', 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd},
	},
	{true, Packet{base.FrameBinary, base.MESSAGE, []byte("hello 你好")},
		[]byte{0x01, 0x01, 0x03, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd},
	},

	{false, Packet{base.FrameString, base.OPEN, []byte{}},
		[]byte("1:0")},
	{false, Packet{base.FrameString, base.MESSAGE, []byte("hello 你好")},
		[]byte("13:4hello 你好")},
	{false, Packet{base.FrameBinary, base.MESSAGE, []byte("hello 你好")},
		[]byte("18:b4aGVsbG8g5L2g5aW9")},
}
