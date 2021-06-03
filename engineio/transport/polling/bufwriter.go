package polling

import (
	"bytes"
	"io"
	"sync"
)

// bufWriter provides a buffer to write. It could write finished frames to other writer.
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

// Write writes data b to the buffer.
// This method is thread safe.
func (w *bufWriter) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}

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

// WriteByte writes a byte b to the buffer.
// This method is thread safe.
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

// WriteFinishedFrames write finished frames in the buffer to the writer to.
// This method is thread safe.
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
