package payload

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeEncoderWriter struct {
	w         io.Writer
	beginErr  error
	endErr    error
	passinErr error
}

func (w *fakeEncoderWriter) beginWrite() (io.Writer, error) {
	if oe, ok := w.beginErr.(Error); ok && oe.Temporary() {
		w.beginErr = nil
		return nil, oe
	}
	return w.w, w.beginErr
}

func (w *fakeEncoderWriter) endWrite(err error) error {
	w.passinErr = err
	return w.endErr
}

func TestEncoder(t *testing.T) {
	assert := assert.New(t)
	must := require.New(t)
	buf := bytes.NewBuffer(nil)
	w := &fakeEncoderWriter{
		w: buf,
	}

	for _, test := range tests {
		buf.Reset()
		e := encoder{
			supportBinary: test.supportBinary,
			encoderWriter: w,
		}

		fw, err := e.NextWriter(test.packet.ft, test.packet.pt)
		must.Nil(err)
		_, err = fw.Write(test.packet.data)
		must.Nil(err)
		err = fw.Close()
		must.Nil(err)

		assert.Equal(test.data, buf.Bytes())
	}
}

func TestEncoderBeginError(t *testing.T) {
	assert := assert.New(t)
	buf := bytes.NewBuffer(nil)
	w := &fakeEncoderWriter{
		w: buf,
	}
	e := encoder{
		supportBinary: true,
		encoderWriter: w,
	}

	buf.Reset()
	targetErr := newOpError("payload", errPaused)
	w.beginErr = targetErr

	_, err := e.NextWriter(base.FrameBinary, base.OPEN)
	assert.Equal(targetErr, err)
}

type errorWrite struct {
	err error
}

func (w *errorWrite) Write(p []byte) (int, error) {
	return 0, w.err
}

func TestEncoderEndError(t *testing.T) {
	assert := assert.New(t)
	must := require.New(t)
	werr := errors.New("write error")
	w := &fakeEncoderWriter{
		w: &errorWrite{
			err: werr,
		},
	}
	e := encoder{
		supportBinary: true,
		encoderWriter: w,
	}

	targetErr := errors.New("error")
	w.endErr = targetErr

	fw, err := e.NextWriter(base.FrameBinary, base.OPEN)
	must.Nil(err)
	err = fw.Close()
	assert.Equal(targetErr, err)
	assert.Equal(w.passinErr, werr)
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
		encoderWriter: &fakeEncoderWriter{
			w: ioutil.Discard,
		},
	}

	// warm up for memory allocation
	for _, p := range packets {
		w, err := e.NextWriter(p.ft, p.pt)
		must.Nil(err)
		_, err = w.Write(p.data)
		must.Nil(err)
		err = w.Close()
		must.Nil(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range packets {
			w, _ := e.NextWriter(p.ft, p.pt)
			w.Write(p.data)
			w.Close()
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
		encoderWriter: &fakeEncoderWriter{
			w: ioutil.Discard,
		},
	}

	// warm up for memory allocation
	for _, p := range packets {
		w, err := e.NextWriter(p.ft, p.pt)
		must.Nil(err)
		_, err = w.Write(p.data)
		must.Nil(err)
		err = w.Close()
		must.Nil(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range packets {
			w, _ := e.NextWriter(p.ft, p.pt)
			w.Write(p.data)
			w.Close()
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
		encoderWriter: &fakeEncoderWriter{
			w: ioutil.Discard,
		},
	}

	// warm up for memory allocation
	for _, p := range packets {
		w, err := e.NextWriter(p.ft, p.pt)
		must.Nil(err)
		_, err = w.Write(p.data)
		must.Nil(err)
		err = w.Close()
		must.Nil(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range packets {
			w, _ := e.NextWriter(p.ft, p.pt)
			w.Write(p.data)
			w.Close()
		}
	}
}
