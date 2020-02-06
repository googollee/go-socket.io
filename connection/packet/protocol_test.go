package packet

import (
	"io"
	"io/ioutil"
	"testing"

	"github.com/googollee/go-socket.io/connection/base"

	"github.com/stretchr/testify/assert"
)

type Frame struct {
	typ  base.FrameType
	data []byte
}

type Packet struct {
	ft   base.FrameType
	pt   base.PacketType
	data []byte
}

var tests = []struct {
	packets []Packet
	frames  []Frame
}{
	{nil, nil},
	{[]Packet{
		{base.FrameString, base.OPEN, []byte{}},
	}, []Frame{
		{base.FrameString, []byte("0")},
	}},
	{[]Packet{
		{base.FrameString, base.MESSAGE, []byte("hello 你好")},
	}, []Frame{
		{base.FrameString, []byte("4hello 你好")},
	}},
	{[]Packet{
		{base.FrameBinary, base.MESSAGE, []byte("hello 你好")},
	}, []Frame{
		{base.FrameBinary, []byte{0x04, 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd}},
	}},
	{[]Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameBinary, base.MESSAGE, []byte("hello\n")},
		{base.FrameString, base.MESSAGE, []byte("你好\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}, []Frame{
		{base.FrameString, []byte("0")},
		{base.FrameBinary, []byte{0x04, 'h', 'e', 'l', 'l', 'o', '\n'}},
		{base.FrameString, []byte("4你好\n")},
		{base.FrameString, []byte("2probe")},
	}},
	{[]Packet{
		{base.FrameBinary, base.MESSAGE, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{base.FrameString, base.MESSAGE, []byte("hello")},
		{base.FrameString, base.CLOSE, []byte{}},
	}, []Frame{
		{base.FrameBinary, []byte{4, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{base.FrameString, []byte("4hello")},
		{base.FrameString, []byte("1")},
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
	discarder := &fakeDiscardWriter{}
	encoder := NewEncoder(discarder)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, _ := encoder.NextWriter(base.FrameString, base.MESSAGE)
		w.Close()
		w, _ = encoder.NextWriter(base.FrameBinary, base.MESSAGE)
		w.Close()
	}
}

func BenchmarkDecoder(b *testing.B) {
	r := newFakeConstReader()
	decoder := NewDecoder(r)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, fr, _ := decoder.NextReader()
		fr.Close()
		_, _, fr, _ = decoder.NextReader()
		fr.Close()
	}
}
