package payload

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
)

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
		closed := make(chan struct{})
		var err atomic.Value
		var wg sync.WaitGroup
		err.Store(io.EOF)
		r := NewDecoder(closed, &err)

		wg.Add(1)
		go func() {
			defer wg.Done()

			buf := bytes.NewBuffer(test.data)
			typ := base.FrameString
			if test.supportBinary {
				typ = base.FrameBinary
			}
			err := r.FeedIn(typ, buf)
			at.Nil(err)

			close(closed)
		}()

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

		wg.Wait()
	}
}

func TestDecoderMultiPacket(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		supportBinary bool
		packets       []Packet
		data          [][]byte
	}{
		{true, []Packet{
			{base.FrameString, base.OPEN, []byte{}},
			{base.FrameBinary, base.MESSAGE, []byte("hel")},
			{base.FrameString, base.MESSAGE, []byte("你")},
			{base.FrameBinary, base.MESSAGE, []byte("hel")},
			{base.FrameString, base.MESSAGE, []byte("你")},
			{base.FrameString, base.PING, []byte("pro")},
		}, [][]byte{[]byte{
			0x00, 0x01, 0xff, '0',
			0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
			0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
		}, []byte{
			0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
			0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
			0x00, 0x06, 0xff, '2', 'p', 'r', 'o', 'b', 'e',
		}}},

		{false, []Packet{
			{base.FrameString, base.OPEN, []byte{}},
			{base.FrameBinary, base.MESSAGE, []byte("hel")},
			{base.FrameString, base.MESSAGE, []byte("你")},
			{base.FrameBinary, base.MESSAGE, []byte("hel")},
			{base.FrameString, base.MESSAGE, []byte("你")},
			{base.FrameString, base.PING, []byte("pro")},
		}, [][]byte{
			[]byte("1:010:b4aGVsbG8K8:4你好\n"),
			[]byte("10:b4aGVsbG8K8:4你好\n6:2probe"),
		}},
	}

	for _, test := range tests {
		closed := make(chan struct{})
		var err atomic.Value
		var wg sync.WaitGroup
		err.Store(io.EOF)
		r := NewDecoder(closed, &err)

		wg.Add(1)
		go func() {
			defer wg.Done()

			for _, bs := range test.data {
				buf := bytes.NewBuffer(bs)
				typ := base.FrameString
				if test.supportBinary {
					typ = base.FrameBinary
				}
				err := r.FeedIn(typ, buf)
				at.Nil(err)
			}

			close(closed)
		}()

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

		wg.Wait()
	}
}

func TestDecoderSwitchFrameType(t *testing.T) {
	at := assert.New(t)
	type Binary struct {
		typ  base.FrameType
		data []byte
	}
	tests := []struct {
		packets []Packet
		binary  []Binary
	}{
		{[]Packet{
			{base.FrameString, base.OPEN, []byte{}},
			{base.FrameBinary, base.MESSAGE, []byte("hel")},
			{base.FrameString, base.MESSAGE, []byte("你")},
			{base.FrameBinary, base.MESSAGE, []byte("hel")},
			{base.FrameString, base.MESSAGE, []byte("你")},
			{base.FrameString, base.PING, []byte("pro")},
		}, []Binary{
			Binary{base.FrameBinary, []byte{
				0x00, 0x01, 0xff, '0',
				0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
				0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
			}}, Binary{base.FrameString,
				[]byte("10:b4aGVsbG8K8:4你好\n6:2probe"),
			}}},
	}

	for _, test := range tests {
		closed := make(chan struct{})
		var err atomic.Value
		var wg sync.WaitGroup
		err.Store(io.EOF)
		r := NewDecoder(closed, &err)

		wg.Add(1)
		go func() {
			defer wg.Done()

			for _, bin := range test.binary {
				buf := bytes.NewBuffer(bin.data)
				err := r.FeedIn(bin.typ, buf)
				at.Nil(err)
			}

			close(closed)
		}()

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

		wg.Wait()
	}
}

func TestDecoderCloseWhenFeedIn(t *testing.T) {
	at := assert.New(t)
	closed := make(chan struct{})
	var err atomic.Value
	var wg sync.WaitGroup
	err.Store(io.EOF)
	r := NewDecoder(closed, &err)

	wg.Add(1)
	go func() {
		defer wg.Done()

		close(closed)
	}()

	e := r.FeedIn(base.FrameBinary, bytes.NewReader([]byte{0x00, 0x01, 0xff, '0'}))
	at.Equal(io.EOF, e)

	wg.Wait()
}

func TestDecoderCloseWhenFraming(t *testing.T) {
	at := assert.New(t)
	closed := make(chan struct{})
	var err atomic.Value
	var wg sync.WaitGroup
	err.Store(io.EOF)
	r := NewDecoder(closed, &err)

	wg.Add(1)
	go func() {
		defer wg.Done()

		ft, pt, rd, err := r.NextReader()
		at.Nil(err)
		at.Equal(base.FrameString, ft)
		at.Equal(base.OPEN, pt)
		at.NotNil(rd)

		close(closed)
	}()

	e := r.FeedIn(base.FrameBinary, bytes.NewReader([]byte{0x00, 0x01, 0xff, '0'}))
	at.Equal(io.EOF, e)

	wg.Wait()
}

func TestDecoderCloseWhenNextRead(t *testing.T) {
	at := assert.New(t)
	closed := make(chan struct{})
	var err atomic.Value
	var wg sync.WaitGroup
	err.Store(io.EOF)
	r := NewDecoder(closed, &err)

	wg.Add(1)
	go func() {
		defer wg.Done()

		close(closed)
	}()

	_, _, _, e := r.NextReader()
	at.Equal(io.EOF, e)

	wg.Wait()
}

type fakeNonByteReader struct {
	buf *bytes.Buffer
}

func (r *fakeNonByteReader) Read(p []byte) (int, error) {
	return r.buf.Read(p)
}

func TestDecoderNonByteReader(t *testing.T) {
	at := assert.New(t)
	closed := make(chan struct{})
	var err atomic.Value
	var wg sync.WaitGroup
	err.Store(io.EOF)
	r := NewDecoder(closed, &err)
	max := 10

	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < max; i++ {
			ft, pt, rd, err := r.NextReader()
			at.Nil(err)
			at.Equal(base.FrameString, ft)
			at.Equal(base.MESSAGE, pt)
			b, err := ioutil.ReadAll(rd)
			at.Nil(err)
			at.Equal("你好\n", string(b))
		}

		close(closed)
	}()

	for i := 0; i < max; i++ {
		reader := fakeNonByteReader{
			buf: bytes.NewBuffer([]byte{0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n'}),
		}
		e := r.FeedIn(base.FrameBinary, &reader)
		if e == io.EOF {
			break
		}
		at.Nil(e)
	}

	wg.Wait()
}

type readCloser struct {
	once   sync.Once
	closed chan struct{}
	err    error
}

func (r *readCloser) Read(p []byte) (int, error) {
	r.once.Do(func() {
		close(r.closed)
	})
	return 0, r.err
}

func TestDecoderCloseWhenRead(t *testing.T) {
	at := assert.New(t)
	closed := make(chan struct{})
	var err atomic.Value
	var wg sync.WaitGroup
	err.Store(io.EOF)
	r := NewDecoder(closed, &err)

	wg.Add(1)
	go func() {
		defer wg.Done()

		_, _, _, err := r.NextReader()
		at.NotNil(err)
	}()

	targetErr := errors.New("error")
	reader := readCloser{
		closed: closed,
		err:    targetErr,
	}
	e := r.FeedIn(base.FrameBinary, &reader)
	at.Equal(targetErr, e)

	wg.Wait()
}

func TestDecoderTimeout(t *testing.T) {
	at := assert.New(t)
	closed := make(chan struct{})
	var err atomic.Value
	err.Store(io.EOF)
	r := NewDecoder(closed, &err)
	e := r.SetDeadline(time.Now().Add(time.Second))
	at.Nil(e)

	begin := time.Now()
	_, _, _, e = r.NextReader()
	at.Equal(ErrTimeout, e)
	end := time.Now()
	duration := end.Sub(begin)
	at.True(duration > time.Second)

	at.Equal(ErrTimeout, err.Load().(error))
}
