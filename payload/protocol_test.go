package payload

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
)

type Packet struct {
	ft   base.FrameType
	pt   base.PacketType
	data []byte
}

var tests = []struct {
	supportBinary bool
	packets       []Packet
	data          []byte
}{
	{true, nil, nil},
	{true, []Packet{
		{base.FrameString, base.OPEN, []byte{}},
	}, []byte{0x00, 0x01, 0xff, '0'}},
	{true, []Packet{
		{base.FrameString, base.MESSAGE, []byte("hello 你好")},
	}, []byte{0x00, 0x01, 0x03, 0xff, '4', 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd}},
	{true, []Packet{
		{base.FrameBinary, base.MESSAGE, []byte("hello 你好")},
	}, []byte{0x01, 0x01, 0x03, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd}},
	{true, []Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameBinary, base.MESSAGE, []byte("hello\n")},
		{base.FrameString, base.MESSAGE, []byte("你好\n")},
		{base.FrameBinary, base.MESSAGE, []byte("hello\n")},
		{base.FrameString, base.MESSAGE, []byte("你好\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}, []byte{
		0x00, 0x01, 0xff, '0',
		0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
		0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
		0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
		0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
		0x00, 0x06, 0xff, '2', 'p', 'r', 'o', 'b', 'e',
	}},
	{true, []Packet{
		{base.FrameBinary, base.MESSAGE, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{base.FrameString, base.MESSAGE, []byte("hello")},
		{base.FrameString, base.CLOSE, []byte{}},
	}, []byte{
		0x01, 0x01, 0x03, 0xff, 0x04, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
		0x00, 0x06, 0xff, '4', 'h', 'e', 'l', 'l', 'o',
		0x00, 0x01, 0xff, '1',
	}},

	{false, nil, nil},
	{false, []Packet{
		{base.FrameString, base.OPEN, []byte{}},
	}, []byte("1:0")},
	{false, []Packet{
		{base.FrameString, base.MESSAGE, []byte("hello 你好")},
	}, []byte("13:4hello 你好")},
	{false, []Packet{
		{base.FrameBinary, base.MESSAGE, []byte("hello 你好")},
	}, []byte("18:b4aGVsbG8g5L2g5aW9")},
	{false, []Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameBinary, base.MESSAGE, []byte("hello\n")},
		{base.FrameString, base.MESSAGE, []byte("你好\n")},
		{base.FrameBinary, base.MESSAGE, []byte("hello\n")},
		{base.FrameString, base.MESSAGE, []byte("你好\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}, []byte("1:010:b4aGVsbG8K8:4你好\n10:b4aGVsbG8K8:4你好\n6:2probe")},
	{false, []Packet{
		{base.FrameBinary, base.MESSAGE, []byte{0, 1, 2, 3, 4}},
		{base.FrameString, base.MESSAGE, []byte("hello")},
	}, []byte("10:b4AAECAwQ=6:4hello")},
}

type fakeWriter struct {
	supportBinary bool
	data          *bytes.Buffer
}

func newFakeWriter(supportBinary bool) *fakeWriter {
	return &fakeWriter{
		supportBinary: supportBinary,
		data:          bytes.NewBuffer(nil),
	}
}

func (w *fakeWriter) SupportBinary() bool {
	return w.supportBinary
}

func (w *fakeWriter) WriteFrame(head, data []byte) error {
	if _, err := w.data.Write(head); err != nil {
		return err
	}
	if _, err := w.data.Write(data); err != nil {
		return err
	}
	return nil
}

func TestEncoder(t *testing.T) {
	at := assert.New(t)

	for _, test := range tests {
		conn := newFakeWriter(test.supportBinary)
		w := NewEncoder(conn)
		for _, packet := range test.packets {
			fw, err := w.NextWriter(packet.ft, packet.pt)
			at.Nil(err)
			_, err = fw.Write(packet.data)
			at.Nil(err)
			err = fw.Close()
			at.Nil(err)
		}
		at.Equal(test.data, conn.data.Bytes())
	}
}

type fakeReader struct {
	supportBinary bool
	*bytes.Buffer
}

func newFakeReader(supportBinary bool, data []byte) *fakeReader {
	return &fakeReader{
		supportBinary: supportBinary,
		Buffer:        bytes.NewBuffer(data),
	}
}

func (r *fakeReader) SupportBinary() bool {
	return r.supportBinary
}

func TestDecoder(t *testing.T) {
	at := assert.New(t)

	for _, test := range tests {
		conn := newFakeReader(test.supportBinary, test.data)
		r := NewDecoder(conn)
		var packets []Packet
		for {
			ft, pt, fr, err := r.NextReader()
			if err != nil {
				at.Equal(io.EOF, err)
				break
			}
			data, err := ioutil.ReadAll(fr)
			at.Nil(err)
			packet := Packet{
				ft:   ft,
				pt:   pt,
				data: data,
			}
			packets = append(packets, packet)
		}
		at.Equal(test.packets, packets)
	}
}

func TestDecoderPartRead(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		supportBinary bool
		packets       []Packet
		data          []byte
	}{
		{true, []Packet{
			{base.FrameString, base.OPEN, []byte{}},
			{base.FrameBinary, base.MESSAGE, []byte("hel")},
			{base.FrameString, base.MESSAGE, []byte("你")},
			{base.FrameBinary, base.MESSAGE, []byte("hel")},
			{base.FrameString, base.MESSAGE, []byte("你")},
			{base.FrameString, base.PING, []byte("pro")},
		}, []byte{
			0x00, 0x01, 0xff, '0',
			0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
			0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
			0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
			0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
			0x00, 0x06, 0xff, '2', 'p', 'r', 'o', 'b', 'e',
		}},

		{false, []Packet{
			{base.FrameString, base.OPEN, []byte{}},
			{base.FrameBinary, base.MESSAGE, []byte("hel")},
			{base.FrameString, base.MESSAGE, []byte("你")},
			{base.FrameBinary, base.MESSAGE, []byte("hel")},
			{base.FrameString, base.MESSAGE, []byte("你")},
			{base.FrameString, base.PING, []byte("pro")},
		}, []byte("1:010:b4aGVsbG8K8:4你好\n10:b4aGVsbG8K8:4你好\n6:2probe")},
	}

	for _, test := range tests {
		conn := newFakeReader(test.supportBinary, test.data)
		r := NewDecoder(conn)
		var packets []Packet
		for {
			ft, pt, fr, err := r.NextReader()
			if err != nil {
				at.Equal(io.EOF, err)
				break
			}
			var data [3]byte
			n, err := io.ReadFull(fr, data[:])
			if err == io.EOF {
				n = 0
			} else {
				at.Nil(err)
			}
			packet := Packet{
				ft:   ft,
				pt:   pt,
				data: data[:n],
			}
			packets = append(packets, packet)
		}
		at.Equal(test.packets, packets)
	}
}

type discarder bool

func (d discarder) SupportBinary() bool {
	return bool(d)
}

func (d discarder) WriteFrame(head, data []byte) error {
	return nil
}

func BenchmarkStringEncoder(b *testing.B) {
	packets := []Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameString, base.MESSAGE, []byte("你好\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}
	w := discarder(false)
	encoder := NewEncoder(w)
	for _, p := range packets {
		w, _ := encoder.NextWriter(p.ft, p.pt)
		w.Write(p.data)
		w.Close()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range packets {
			w, _ := encoder.NextWriter(p.ft, p.pt)
			w.Write(p.data)
			w.Close()
		}
	}
}

func BenchmarkB64Encoder(b *testing.B) {
	packets := []Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameBinary, base.MESSAGE, []byte("hello\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}
	w := discarder(false)
	encoder := NewEncoder(w)
	for _, p := range packets {
		w, _ := encoder.NextWriter(p.ft, p.pt)
		w.Write(p.data)
		w.Close()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range packets {
			w, _ := encoder.NextWriter(p.ft, p.pt)
			w.Write(p.data)
			w.Close()
		}
	}
}

func BenchmarkBinaryEncoder(b *testing.B) {
	packets := []Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameBinary, base.MESSAGE, []byte("hello\n")},
		{base.FrameString, base.MESSAGE, []byte("你好\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}
	w := discarder(true)
	encoder := NewEncoder(w)
	for _, p := range packets {
		w, _ := encoder.NextWriter(p.ft, p.pt)
		w.Write(p.data)
		w.Close()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range packets {
			w, _ := encoder.NextWriter(p.ft, p.pt)
			w.Write(p.data)
			w.Close()
		}
	}
}

func BenchmarkStringDecoder(b *testing.B) {
	data := bytes.Repeat([]byte("1:08:4你好\n6:2probe"), b.N)
	conn := newFakeReader(false, data)
	decoder := NewDecoder(conn)
	buf := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 3; j++ {
			_, _, r, _ := decoder.NextReader()
			r.Read(buf)
		}
	}
}

func BenchmarkB64Decoder(b *testing.B) {
	data := bytes.Repeat([]byte("1:010:b4aGVsbG8K6:2probe"), b.N)
	conn := newFakeReader(false, data)
	decoder := NewDecoder(conn)
	buf := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 3; j++ {
			_, _, r, _ := decoder.NextReader()
			r.Read(buf)
		}
	}
}

func BenchmarkBinaryDecoder(b *testing.B) {
	data := bytes.Repeat([]byte{
		0x00, 0x01, 0xff, '0',
		0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
		0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
		0x00, 0x06, 0xff, '2', 'p', 'r', 'o', 'b', 'e',
	}, b.N)
	conn := newFakeReader(true, data)
	decoder := NewDecoder(conn)
	buf := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 4; j++ {
			_, _, r, _ := decoder.NextReader()
			r.Read(buf)
		}
	}
}
