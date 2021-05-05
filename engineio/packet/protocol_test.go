package packet

import (
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Frame struct {
	typ  FrameType
	data []byte
}

type Packet struct {
	ft   FrameType
	pt   PacketType
	data []byte
}

var tests = []struct {
	packets []Packet
	frames  []Frame
}{
	{nil, nil},
	{[]Packet{
		{FrameString, OPEN, []byte{}},
	}, []Frame{
		{FrameString, []byte("0")},
	}},
	{[]Packet{
		{FrameString, MESSAGE, []byte("hello 你好")},
	}, []Frame{
		{FrameString, []byte("4hello 你好")},
	}},
	{[]Packet{
		{FrameBinary, MESSAGE, []byte("hello 你好")},
	}, []Frame{
		{FrameBinary, []byte{0x04, 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd}},
	}},
	{[]Packet{
		{FrameString, OPEN, []byte{}},
		{FrameBinary, MESSAGE, []byte("hello\n")},
		{FrameString, MESSAGE, []byte("你好\n")},
		{FrameString, PING, []byte("probe")},
	}, []Frame{
		{FrameString, []byte("0")},
		{FrameBinary, []byte{0x04, 'h', 'e', 'l', 'l', 'o', '\n'}},
		{FrameString, []byte("4你好\n")},
		{FrameString, []byte("2probe")},
	}},
	{[]Packet{
		{FrameBinary, MESSAGE, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{FrameString, MESSAGE, []byte("hello")},
		{FrameString, CLOSE, []byte{}},
	}, []Frame{
		{FrameBinary, []byte{4, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{FrameString, []byte("4hello")},
		{FrameString, []byte("1")},
	}},
}

func TestEncoder(t *testing.T) {
	at := assert.New(t)

	for _, test := range tests {
		w := newFakeConnWriter()
		encoder := NewEncoder(w)
		for _, p := range test.packets {
			fw, err := encoder.NextWriter(p.ft, p.pt)
			at.Nil(err)
			_, err = fw.Write(p.data)
			at.Nil(err)
			err = fw.Close()
			at.Nil(err)
		}
		at.Equal(test.frames, w.frames)
	}
}

func TestDecoder(t *testing.T) {
	at := assert.New(t)

	for _, test := range tests {
		r := newFakeConnReader(test.frames)
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
				ft:   ft,
				pt:   pt,
				data: b,
			})
		}
		at.Equal(test.packets, output)
	}
}

func BenchmarkEncoder(b *testing.B) {
	encoder := NewEncoder(&fakeDiscardWriter{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, _ := encoder.NextWriter(FrameString, MESSAGE)
		w.Close()

		w, _ = encoder.NextWriter(FrameBinary, MESSAGE)
		w.Close()
	}
}

func BenchmarkDecoder(b *testing.B) {
	decoder := NewDecoder(newFakeConstReader())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, fr, _ := decoder.NextReader()
		fr.Close()
		_, _, fr, _ = decoder.NextReader()
		fr.Close()
	}
}
