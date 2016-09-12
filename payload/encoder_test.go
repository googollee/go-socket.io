package payload

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
)

func TestEncoderCloseWhileFraming(t *testing.T) {
	at := assert.New(t)
	sig := NewSignal()

	w := NewEncoder(true, sig)

	wr, e := w.NextWriter(base.FrameString, base.OPEN)
	at.Nil(e)
	sig.Close()
	e = wr.Close()
	at.Equal(io.EOF, e)
}

type errWriter struct {
	err error
	sig *Signal
}

func (w *errWriter) Write(p []byte) (int, error) {
	if w.err == nil {
		w.sig.Close()
		return len(p), nil
	}
	return 0, w.err
}

func TestEncoderCloseWhenWriting(t *testing.T) {
	at := assert.New(t)
	sig := NewSignal()
	var wg sync.WaitGroup

	w := NewEncoder(true, sig)

	wg.Add(1)
	go func() {
		defer wg.Done()
		wr, e := w.NextWriter(base.FrameString, base.OPEN)
		at.Nil(e)
		e = wr.Close()
		at.Equal(io.EOF, e)
	}()

	writer := errWriter{
		sig: sig,
	}
	e := w.FlushOut(&writer)
	at.Equal(io.EOF, e)

	wg.Wait()
}

func TestEncoderErrorWhenWriting(t *testing.T) {
	at := assert.New(t)
	sig := NewSignal()
	var wg sync.WaitGroup

	w := NewEncoder(true, sig)

	writer := errWriter{
		sig: sig,
		err: errors.New("error"),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		wr, e := w.NextWriter(base.FrameString, base.OPEN)
		at.Nil(e)
		e = wr.Close()
		at.Equal(writer.err, e)
	}()

	e := w.FlushOut(&writer)
	at.Equal(writer.err, e)

	wg.Wait()
}

func TestEncoderTimeout(t *testing.T) {
	at := assert.New(t)
	sig := NewSignal()

	w := NewEncoder(true, sig)

	e := w.SetDeadline(time.Now().Add(time.Second))
	at.Nil(e)

	begin := time.Now()
	wr, e := w.NextWriter(base.FrameString, base.OPEN)
	at.Nil(e)
	e = wr.Close()
	at.Equal(ErrTimeout, e)
	end := time.Now()
	at.True(end.Sub(begin) > time.Second)

	at.Equal(ErrTimeout, sig.LoadError().(error))
}

func TestEncoderPauseBinary(t *testing.T) {
	assert := assert.New(t)
	sig := NewSignal()
	w := newEncoder(true, sig)
	buf := bytes.NewBuffer(nil)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := w.FlushOut(buf)
		assert.Nil(err)
	}()

	sig.Pause()
	wg.Wait()
	assert.Equal([]byte{0x0, 0x1, 0xff, 0x36}, buf.Bytes())
}

func TestEncoderPauseText(t *testing.T) {
	assert := assert.New(t)
	sig := NewSignal()
	w := newEncoder(false, sig)
	buf := bytes.NewBuffer(nil)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := w.FlushOut(buf)
		assert.Nil(err)
	}()

	sig.Pause()
	wg.Wait()

	assert.Equal([]byte("1:6"), buf.Bytes())
}
