package payload

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"
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

type nonByteWriter struct {
	buf *bytes.Buffer
}

func (w *nonByteWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func TestEncoder(t *testing.T) {
	at := assert.New(t)

	for _, test := range tests {
		closed := make(chan struct{})
		var err AtomicError
		var wg sync.WaitGroup

		w := NewEncoder(test.supportBinary, closed, &err)
		writer := &nonByteWriter{
			buf: bytes.NewBuffer(nil),
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				err := w.FlushOut(writer)
				if err != nil {
					at.Equal(io.EOF, err)
					return
				}
			}
		}()

		fw, e := w.NextWriter(test.packet.ft, test.packet.pt)
		at.Nil(e)
		_, e = fw.Write(test.packet.data)
		at.Nil(e)
		e = fw.Close()
		at.Nil(e)

		close(closed)
		wg.Wait()

		at.Equal(test.data, writer.buf.Bytes())
	}
}

func TestDecoder(t *testing.T) {
	at := assert.New(t)

	for _, test := range tests {
		closed := make(chan struct{})
		var err AtomicError
		var wg sync.WaitGroup

		r := NewDecoder(closed, &err)
		var packets []Packet

		wg.Add(1)
		go func() {
			defer wg.Done()

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
		}()

		typ := base.FrameString
		if test.supportBinary {
			typ = base.FrameBinary
		}

		e := r.FeedIn(typ, bytes.NewReader(test.data))
		at.Nil(e)

		close(closed)
		wg.Wait()

		at.Equal([]Packet{test.packet}, packets)
	}
}

type discarder struct{}

func (d discarder) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d discarder) WriteByte(byte) error {
	return nil
}

func BenchmarkStringEncoder(b *testing.B) {
	packets := []Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameString, base.MESSAGE, []byte("你好\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}
	closed := make(chan struct{})
	var err AtomicError
	var wg sync.WaitGroup
	encoder := NewEncoder(false, closed, &err)
	writer := discarder{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			if err := encoder.FlushOut(writer); err != nil {
				return
			}
		}
	}()

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

	b.StopTimer()

	close(closed)
	wg.Wait()
}

func BenchmarkB64Encoder(b *testing.B) {
	packets := []Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameBinary, base.MESSAGE, []byte("hello\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}
	closed := make(chan struct{})
	var err AtomicError
	var wg sync.WaitGroup
	encoder := NewEncoder(false, closed, &err)
	writer := discarder{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			if err := encoder.FlushOut(writer); err != nil {
				return
			}
		}
	}()

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

	b.StopTimer()

	close(closed)
	wg.Wait()
}

func BenchmarkBinaryEncoder(b *testing.B) {
	packets := []Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameBinary, base.MESSAGE, []byte("hello\n")},
		{base.FrameString, base.MESSAGE, []byte("你好\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}
	closed := make(chan struct{})
	var err AtomicError
	var wg sync.WaitGroup
	encoder := NewEncoder(true, closed, &err)
	writer := discarder{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			if err := encoder.FlushOut(writer); err != nil {
				return
			}
		}
	}()

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

	b.StopTimer()

	close(closed)
	wg.Wait()
}

func BenchmarkStringDecoder(b *testing.B) {
	data := bytes.Repeat([]byte("1:08:4你好\n6:2probe"), b.N)
	reader := bytes.NewReader(data)
	closed := make(chan struct{})
	var err AtomicError
	var wg sync.WaitGroup
	decoder := NewDecoder(closed, &err)

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := decoder.FeedIn(base.FrameString, reader)
		if err != nil {
			return
		}
	}()

	buf := make([]byte, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 3; j++ {
			_, _, r, _ := decoder.NextReader()
			r.Read(buf)
		}
	}

	b.StopTimer()

	close(closed)
	wg.Wait()
}

func BenchmarkB64Decoder(b *testing.B) {
	data := bytes.Repeat([]byte("1:010:b4aGVsbG8K6:2probe"), b.N)
	reader := bytes.NewReader(data)
	closed := make(chan struct{})
	var err AtomicError
	var wg sync.WaitGroup
	decoder := NewDecoder(closed, &err)

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			err := decoder.FeedIn(base.FrameString, reader)
			if err != nil {
				return
			}
		}
	}()

	buf := make([]byte, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 3; j++ {
			_, _, r, _ := decoder.NextReader()
			r.Read(buf)
		}
	}

	b.StopTimer()

	close(closed)
	wg.Wait()
}

func BenchmarkBinaryDecoder(b *testing.B) {
	data := bytes.Repeat([]byte{
		0x00, 0x01, 0xff, '0',
		0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
		0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
		0x00, 0x06, 0xff, '2', 'p', 'r', 'o', 'b', 'e',
	}, b.N)
	reader := bytes.NewReader(data)
	closed := make(chan struct{})
	var err AtomicError
	var wg sync.WaitGroup
	decoder := NewDecoder(closed, &err)

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			err := decoder.FeedIn(base.FrameBinary, reader)
			if err != nil {
				return
			}
		}
	}()

	buf := make([]byte, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 4; j++ {
			_, _, r, _ := decoder.NextReader()
			r.Read(buf)
		}
	}

	b.StopTimer()

	close(closed)
	wg.Wait()
}

func TestAtomicError(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		storeErr error
		loadErr  error
	}{
		{nil, io.EOF},
		{ErrTimeout, ErrTimeout},
	}
	for _, test := range tests {
		var err AtomicError
		at.Equal(io.EOF, err.Load())
		at.Equal(test.storeErr, err.Store(test.storeErr))
		at.Equal(test.loadErr, err.Load())
	}
}
