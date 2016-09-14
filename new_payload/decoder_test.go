package payload

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecoder(t *testing.T) {
	assert := assert.New(t)
	must := require.New(t)

	for _, test := range tests {
		d := decoder{}
		d.FeedIn(bytes.NewReader(test.data), test.supportBinary)
		var packets []Packet

		for {
			ft, pt, fr, err := d.NextReader()
			if err != nil {
				must.Equal(io.EOF, err)
				break
			}
			data, err := ioutil.ReadAll(fr)
			must.Nil(err)
			packet := Packet{
				ft:   ft,
				pt:   pt,
				data: data,
			}
			packets = append(packets, packet)
		}

		assert.Equal([]Packet{test.packet}, packets)
	}
}

type fakeReader struct {
	r   io.Reader
	err error
}

func newFakeReader(data []byte) *fakeReader {
	return &fakeReader{
		r: bytes.NewReader(data),
	}
}

func (r *fakeReader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.r.Read(p)
}

func TestDecoderNonByteReader(t *testing.T) {
	assert := assert.New(t)
	must := require.New(t)

	for _, test := range tests {
		d := decoder{}
		d.FeedIn(newFakeReader(test.data), test.supportBinary)
		var packets []Packet

		for {
			ft, pt, fr, err := d.NextReader()
			if err != nil {
				must.Equal(io.EOF, err)
				break
			}
			data, err := ioutil.ReadAll(fr)
			must.Nil(err)
			packet := Packet{
				ft:   ft,
				pt:   pt,
				data: data,
			}
			packets = append(packets, packet)
		}

		assert.Equal([]Packet{test.packet}, packets)
	}
}

func TestDecoderNextReaderError(t *testing.T) {
	assert := assert.New(t)

	d := decoder{}
	r := newFakeReader([]byte{0x00, 0x01, 0xff, '0'})
	d.FeedIn(r, true)

	targetErr := errors.New("error")
	r.err = targetErr
	_, _, _, err := d.NextReader()
	assert.Equal(targetErr, err)
}

type fakeErrorReader struct {
	r   *bytes.Buffer
	err error
}

func newFakeErrorReader(data []byte) *fakeErrorReader {
	return &fakeErrorReader{
		r: bytes.NewBuffer(data),
	}
}

func (r *fakeErrorReader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.r.Read(p)
}

func (r *fakeErrorReader) ReadByte() (byte, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.r.ReadByte()
}

func TestDecoderReadError(t *testing.T) {
	assert := assert.New(t)
	must := require.New(t)
	targetErr := errors.New("error")

	var tests = []struct {
		supportBinary bool
		packet        Packet
		data          []byte
	}{
		{true, Packet{base.FrameString, base.MESSAGE, []byte("hello 你好")},
			[]byte{0x00, 0x01, 0x03, 0xff, '4', 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd},
		},
		{true, Packet{base.FrameBinary, base.MESSAGE, []byte("hello 你好")},
			[]byte{0x01, 0x01, 0x03, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', ' ', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd},
		},

		{false, Packet{base.FrameString, base.MESSAGE, []byte("hello 你好")},
			[]byte("13:4hello 你好")},
		{false, Packet{base.FrameBinary, base.MESSAGE, []byte("hello 你好")},
			[]byte("18:b4aGVsbG8g5L2g5aW9")},
	}

	for _, test := range tests {
		d := decoder{}
		r := newFakeErrorReader(test.data)
		d.FeedIn(r, test.supportBinary)

		_, _, fr, err := d.NextReader()
		if err != nil {
			must.Equal(io.EOF, err)
			break
		}

		r.err = targetErr
		_, err = ioutil.ReadAll(fr)
		assert.Equal(targetErr, err)
	}
}

func BenchmarkStringDecoder(b *testing.B) {
	data := bytes.Repeat([]byte("8:4你好\n6:2probe"), b.N+1)
	must := require.New(b)
	reader := bytes.NewReader(data)
	d := decoder{}
	d.FeedIn(reader, false)
	buf := make([]byte, 10)

	// check
	for j := 0; j < 2; j++ {
		_, _, r, err := d.NextReader()
		must.Nil(err)
		_, err = r.Read(buf)
		must.Nil(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 2; j++ {
			_, _, r, _ := d.NextReader()
			r.Read(buf)
		}
	}
}

func BenchmarkB64Decoder(b *testing.B) {
	data := bytes.Repeat([]byte("10:b4aGVsbG8K6:2probe"), b.N+1)
	must := require.New(b)
	reader := bytes.NewReader(data)
	d := decoder{}
	d.FeedIn(reader, false)
	buf := make([]byte, 10)

	// check
	for j := 0; j < 2; j++ {
		_, _, r, err := d.NextReader()
		must.Nil(err)
		_, err = r.Read(buf)
		must.Nil(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 2; j++ {
			_, _, r, _ := d.NextReader()
			r.Read(buf)
		}
	}
}

func BenchmarkBinaryDecoder(b *testing.B) {
	data := bytes.Repeat([]byte{
		0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
		0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
		0x00, 0x06, 0xff, '2', 'p', 'r', 'o', 'b', 'e',
	}, b.N+1)
	must := require.New(b)
	reader := bytes.NewReader(data)
	d := decoder{}
	d.FeedIn(reader, true)
	buf := make([]byte, 10)

	// check
	for j := 0; j < 3; j++ {
		_, _, r, err := d.NextReader()
		must.Nil(err)
		_, err = r.Read(buf)
		must.Nil(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 3; j++ {
			_, _, r, _ := d.NextReader()
			r.Read(buf)
		}
	}
}
