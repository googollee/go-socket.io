package packet

import (
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/googollee/go-socket.io/engineio/frame"
)

var tests = []struct {
	packets []Packet
	frames  []Frame
}{
	{nil, nil},
	{[]Packet{
		{frame.String, OPEN, []byte{}},
	}, []Frame{
		{frame.String, []byte("0")},
	},
	},
	{[]Packet{
		{frame.String, MESSAGE, []byte("hello 你好")},
	}, []Frame{
		{frame.String, []byte("4hello 你好")},
	},
	},
	{[]Packet{
		{frame.Binary, MESSAGE, []byte("hello 你好")},
	}, []Frame{
		{frame.Binary, []byte{0x04, 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd}},
	},
	},
	{[]Packet{
		{frame.String, OPEN, []byte{}},
		{frame.Binary, MESSAGE, []byte("hello\n")},
		{frame.String, MESSAGE, []byte("你好\n")},
		{frame.String, PING, []byte("probe")},
	}, []Frame{
		{frame.String, []byte("0")},
		{frame.Binary, []byte{0x04, 'h', 'e', 'l', 'l', 'o', '\n'}},
		{frame.String, []byte("4你好\n")},
		{frame.String, []byte("2probe")},
	},
	},
	{[]Packet{
		{frame.Binary, MESSAGE, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{frame.String, MESSAGE, []byte("hello")},
		{frame.String, CLOSE, []byte{}},
	}, []Frame{
		{frame.Binary, []byte{4, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{frame.String, []byte("4hello")},
		{frame.String, []byte("1")},
	},
	},
}

func TestDecoder(t *testing.T) {
	at := assert.New(t)

	for _, test := range tests {
		r := NewFakeConnReader(test.frames)
		decoder := NewDecoder(r)
		var output []Packet
		for {
			ft, pt, fr, err := decoder.NextReader()
			if err != nil {
				at.Equal(io.EOF, err)
				break
			}
			b, err := ioutil.ReadAll(fr)
			at.Nil(err)
			fr.Close()

			output = append(output, Packet{
				FType: ft,
				PType: pt,
				Data:  b,
			})
		}
		at.Equal(test.packets, output)
	}
}

func BenchmarkDecoder(b *testing.B) {
	decoder := NewDecoder(NewFakeConstReader())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, fr, _ := decoder.NextReader()
		fr.Close()
		_, _, fr, _ = decoder.NextReader()
		fr.Close()
	}
}
