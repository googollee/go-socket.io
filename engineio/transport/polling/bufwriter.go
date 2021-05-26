package polling

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

var ErrNoSpace = errors.New("no enough space to write")

type bufWriter struct {
	locker sync.Mutex
	data   []byte
	length int
}

func newBufWriter(buf []byte) *bufWriter {
	return &bufWriter{
		data: buf,
	}
}

func (w *bufWriter) Write(b []byte) (int, error) {
	w.locker.Lock()
	defer w.locker.Unlock()

	l := len(w.data) - w.length
	if l > len(b) {
		l = len(b)
	}
	if l == 0 {
		return 0, ErrNoSpace
	}

	copy(w.data[w.length:], b[:l])
	w.length += l

	return l, nil
}

func (w *bufWriter) WriteByte(b byte) error {
	w.locker.Lock()
	defer w.locker.Unlock()

	l := len(w.data) - w.length
	if l == 0 {
		return ErrNoSpace
	}

	w.data[w.length] = b
	w.length++

	return nil
}

func (w *bufWriter) WriteFinishedFrames(to io.Writer) (int, error) {
	w.locker.Lock()
	defer w.locker.Unlock()

	end := bytes.LastIndexByte(w.data[:w.length], separator)
	if end < 0 {
		// the finished frame must have a separator at the end.
		return 0, nil
	}

	start := 0
	for start < end {
		n, err := to.Write(w.data[start:end])
		start += n
		if err != nil {
			break
		}
	}

	if w.data[start] == separator {
		// ignore the last separator.
		start++
	}
	copy(w.data, w.data[start:])
	w.length -= start

	return start, nil
}
