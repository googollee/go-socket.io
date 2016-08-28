package payload

import (
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
	closed := make(chan struct{})
	var err AtomicError

	w := NewEncoder(true, closed, &err)

	wr, e := w.NextWriter(base.FrameString, base.OPEN)
	at.Nil(e)
	close(closed)
	e = wr.Close()
	at.Equal(io.EOF, e)
}

type errWriter struct {
	err    error
	closed chan struct{}
}

func (w *errWriter) Write(p []byte) (int, error) {
	if w.err == nil {
		close(w.closed)
		return len(p), nil
	}
	return 0, w.err
}

func TestEncoderCloseWhenWriting(t *testing.T) {
	at := assert.New(t)
	closed := make(chan struct{})
	var err AtomicError
	var wg sync.WaitGroup

	w := NewEncoder(true, closed, &err)

	wg.Add(1)
	go func() {
		defer wg.Done()
		wr, e := w.NextWriter(base.FrameString, base.OPEN)
		at.Nil(e)
		e = wr.Close()
		at.Equal(io.EOF, e)
	}()

	writer := errWriter{
		closed: closed,
	}
	e := w.FlushOut(&writer)
	at.Equal(io.EOF, e)

	wg.Wait()
}

func TestEncoderErrorWhenWriting(t *testing.T) {
	at := assert.New(t)
	closed := make(chan struct{})
	var err AtomicError
	var wg sync.WaitGroup

	w := NewEncoder(true, closed, &err)

	writer := errWriter{
		closed: closed,
		err:    errors.New("error"),
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
	closed := make(chan struct{})
	var err AtomicError

	w := NewEncoder(true, closed, &err)

	e := w.SetDeadline(time.Now().Add(time.Second))
	at.Nil(e)

	begin := time.Now()
	wr, e := w.NextWriter(base.FrameString, base.OPEN)
	at.Nil(e)
	e = wr.Close()
	at.Equal(ErrTimeout, e)
	end := time.Now()
	at.True(end.Sub(begin) > time.Second)

	at.Equal(ErrTimeout, err.Load().(error))
}
