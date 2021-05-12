package payload

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReader struct {
	r io.Reader
}

func (r fakeReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

type fakeReaderFeeder struct {
	data          []byte
	supportBinary bool
	returnError   error
	sendError     error
	getCounter    int
	putCounter    int
}

func (f *fakeReaderFeeder) getReader() (io.Reader, bool, error) {
	f.getCounter++
	return fakeReader{bytes.NewReader(f.data)}, f.supportBinary, f.returnError
}

func (f *fakeReaderFeeder) putReader(err error) error {
	f.putCounter++
	f.sendError = err
	return f.returnError
}

func TestDecoder(t *testing.T) {
	assert := assert.New(t)
	must := require.New(t)

	for _, test := range tests {
		feeder := fakeReaderFeeder{
			data:          test.data,
			supportBinary: test.supportBinary,
		}
		d := decoder{
			feeder: &feeder,
		}
		var packets []Packet

		for i := 0; i < len(test.packets); i++ {
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
			err = fr.Close()
			must.Nil(err)
			packets = append(packets, packet)
		}

		assert.Equal(test.packets, packets)
		assert.Equal(feeder.getCounter, 1)
		assert.Equal(feeder.putCounter, 1)
	}
}

func TestDecoderNextReaderError(t *testing.T) {
	assert := assert.New(t)

	feeder := fakeReaderFeeder{
		data:          []byte{0x00, 0x01, 0xff, '0'},
		supportBinary: true,
	}
	d := decoder{
		feeder: &feeder,
	}

	targetErr := errors.New("error")
	feeder.returnError = targetErr
	_, _, _, err := d.NextReader()
	assert.Equal(targetErr, err)
}

func BenchmarkStringDecoder(b *testing.B) {
	feeder := fakeReaderFeeder{
		data:          []byte("8:4你好\n6:2probe"),
		supportBinary: false,
	}
	d := decoder{
		feeder: &feeder,
	}
	buf := make([]byte, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 2; j++ {
			_, _, r, _ := d.NextReader()
			r.Read(buf)
			r.Close()
		}
	}
}

func BenchmarkB64Decoder(b *testing.B) {
	feeder := fakeReaderFeeder{
		data:          []byte("10:b4aGVsbG8K6:2probe"),
		supportBinary: false,
	}
	d := decoder{
		feeder: &feeder,
	}
	buf := make([]byte, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 2; j++ {
			_, _, r, _ := d.NextReader()
			r.Read(buf)
			r.Close()
		}
	}
}

func BenchmarkBinaryDecoder(b *testing.B) {
	feeder := fakeReaderFeeder{
		data: []byte{
			0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
			0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
			0x00, 0x06, 0xff, '2', 'p', 'r', 'o', 'b', 'e',
		},
		supportBinary: true,
	}
	d := decoder{
		feeder: &feeder,
	}
	buf := make([]byte, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 3; j++ {
			_, _, r, _ := d.NextReader()
			r.Read(buf)
			r.Close()
		}
	}
}
