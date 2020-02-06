package payload

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/googollee/go-socket.io/connection/base"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWriterFeeder struct {
	w           io.Writer
	returnError error
	passinErr   error
}

func (f *fakeWriterFeeder) getWriter() (io.Writer, error) {
	if oe, ok := f.returnError.(Error); ok && oe.Temporary() {
		f.returnError = nil
		return nil, oe
	}
	return f.w, f.returnError
}

func (f *fakeWriterFeeder) putWriter(err error) error {
	f.passinErr = err
	return f.returnError
}

func TestEncoder(t *testing.T) {
	assert := assert.New(t)
	must := require.New(t)
	buf := bytes.NewBuffer(nil)
	f := &fakeWriterFeeder{
		w: buf,
	}

	for _, test := range tests {
		buf.Reset()
		e := encoder{
			supportBinary: test.supportBinary,
			feeder:        f,
		}

		for _, packet := range test.packets {
			fw, err := e.NextWriter(packet.ft, packet.pt)
			must.Nil(err)
			_, err = fw.Write(packet.data)
			must.Nil(err)
			err = fw.Close()
			must.Nil(err)
		}

		assert.Equal(test.data, buf.Bytes())
	}
}

func TestEncoderBeginError(t *testing.T) {
	assert := assert.New(t)
	buf := bytes.NewBuffer(nil)
	f := &fakeWriterFeeder{
		w: buf,
	}
	e := encoder{
		supportBinary: true,
		feeder:        f,
	}

	buf.Reset()
	targetErr := newOpError("payload", errPaused)
	f.returnError = targetErr

	_, err := e.NextWriter(base.FrameBinary, base.OPEN)
	assert.Equal(targetErr, err)
}

type errorWrite struct {
	err error
}

func (f *errorWrite) Write(p []byte) (int, error) {
	return 0, f.err
}

func TestEncoderEndError(t *testing.T) {
	assert := assert.New(t)
	must := require.New(t)
	werr := errors.New("write error")
	f := &fakeWriterFeeder{
		w: &errorWrite{
			err: werr,
		},
	}
	e := encoder{
		supportBinary: true,
		feeder:        f,
	}

	targetErr := errors.New("error")

	fw, err := e.NextWriter(base.FrameBinary, base.OPEN)
	must.Nil(err)
	f.returnError = targetErr
	err = fw.Close()
	assert.Equal(targetErr, err)
	assert.Equal(f.passinErr, werr)
}

func TestEncoderNOOP(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		supportBinary bool
		data          []byte
	}{
		{true, []byte{0x00, 0x01, 0xff, '6'}},
		{false, []byte("1:6")},
	}

	for _, test := range tests {
		e := encoder{
			supportBinary: test.supportBinary,
		}
		assert.Equal(test.data, e.NOOP())
	}

	// NOOP should be thread-safe
	var wg sync.WaitGroup
	max := 100
	wg.Add(100)
	for i := 0; i < max; i++ {
		go func(i int) {
			defer wg.Done()
			e := encoder{
				supportBinary: i&0x1 == 0,
			}
			e.NOOP()
		}(i)
	}
	wg.Wait()
}

func BenchmarkStringEncoder(b *testing.B) {
	must := require.New(b)
	packets := []Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameString, base.MESSAGE, []byte("你好\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}
	e := encoder{
		supportBinary: false,
		feeder: &fakeWriterFeeder{
			w: ioutil.Discard,
		},
	}

	// warm up for memory allocation
	for _, p := range packets {
		f, err := e.NextWriter(p.ft, p.pt)
		must.Nil(err)
		_, err = f.Write(p.data)
		must.Nil(err)
		err = f.Close()
		must.Nil(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range packets {
			f, _ := e.NextWriter(p.ft, p.pt)
			f.Write(p.data)
			f.Close()
		}
	}
}

func BenchmarkB64Encoder(b *testing.B) {
	must := require.New(b)
	packets := []Packet{
		{base.FrameBinary, base.OPEN, []byte{}},
		{base.FrameBinary, base.MESSAGE, []byte("你好\n")},
		{base.FrameBinary, base.PING, []byte("probe")},
	}
	e := encoder{
		supportBinary: false,
		feeder: &fakeWriterFeeder{
			w: ioutil.Discard,
		},
	}

	// warm up for memory allocation
	for _, p := range packets {
		f, err := e.NextWriter(p.ft, p.pt)
		must.Nil(err)
		_, err = f.Write(p.data)
		must.Nil(err)
		err = f.Close()
		must.Nil(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range packets {
			f, _ := e.NextWriter(p.ft, p.pt)
			f.Write(p.data)
			f.Close()
		}
	}
}

func BenchmarkBinaryEncoder(b *testing.B) {
	must := require.New(b)
	packets := []Packet{
		{base.FrameString, base.OPEN, []byte{}},
		{base.FrameBinary, base.MESSAGE, []byte("你好\n")},
		{base.FrameString, base.PING, []byte("probe")},
	}
	e := encoder{
		supportBinary: true,
		feeder: &fakeWriterFeeder{
			w: ioutil.Discard,
		},
	}

	// warm up for memory allocation
	for _, p := range packets {
		f, err := e.NextWriter(p.ft, p.pt)
		must.Nil(err)
		_, err = f.Write(p.data)
		must.Nil(err)
		err = f.Close()
		must.Nil(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range packets {
			f, _ := e.NextWriter(p.ft, p.pt)
			f.Write(p.data)
			f.Close()
		}
	}
}
