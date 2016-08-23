package payload

import (
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
)

func TestEncoderCloseWhileFraming(t *testing.T) {
	at := assert.New(t)
	closed := make(chan struct{})
	var err atomic.Value

	err.Store(io.EOF)
	w := NewEncoder(true, closed, &err)

	wr, e := w.NextWriter(base.FrameString, base.OPEN)
	at.Nil(e)
	close(closed)
	e = wr.Close()
	at.Equal(io.EOF, e)
}

type longWriter struct {
	closed chan struct{}
}

func (w *longWriter) Write(p []byte) (int, error) {
	close(w.closed)
	return len(p), nil
}

func TestEncoderCloseWhenWriting(t *testing.T) {
	at := assert.New(t)
	closed := make(chan struct{})
	var err atomic.Value
	var wg sync.WaitGroup

	err.Store(io.EOF)
	w := NewEncoder(true, closed, &err)

	wg.Add(1)
	go func() {
		defer wg.Done()
		wr, e := w.NextWriter(base.FrameString, base.OPEN)
		at.Nil(e)
		e = wr.Close()
		at.Equal(io.EOF, e)
	}()

	writer := longWriter{
		closed: closed,
	}
	e := w.FlushOut(&writer)
	at.Equal(io.EOF, e)

	wg.Wait()
}
