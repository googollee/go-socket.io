package polling

import (
	"io"
)

// bufReader is similar to bufio.Reader, with a customed buffer.
type bufReader struct {
	buf        []byte
	start, end int
	rd         io.Reader
	finished   bool
}

func newBufReader(buf []byte, rd io.Reader) *bufReader {
	return &bufReader{
		buf: buf,
		rd:  rd,
	}
}

// Fill reads data from underline reader to the buffer.
func (r *bufReader) Fill() error {
	if r.start != r.end {
		return nil
	}

	r.start = 0
	var err error
	r.end, err = r.rd.Read(r.buf)
	if err == io.EOF && !r.finished {
		// Ignore first io.EOF and return nil.
		r.finished = true
		return nil
	}
	return err
}

// Read store bytes to b.
func (r *bufReader) Read(b []byte) (int, error) {
	if err := r.Fill(); err != nil {
		return 0, err
	}

	n := r.end - r.start
	if n > len(b) {
		n = len(b)
	}

	copy(b, r.buf[r.start:r.start+n])
	r.start += n
	return n, nil
}

// PushBack pushes n bytes back to the buffer.
// It can't push back data out of the buffer.
func (r *bufReader) PushBack(n int) error {
	if n > r.start || n < 0 {
		return ErrNoEnoughBuf
	}

	r.start -= n
	return nil
}

// ReadByte reads a byte from buffer.
func (r *bufReader) ReadByte() (byte, error) {
	if err := r.Fill(); err != nil {
		return 0, err
	}

	ret := r.buf[r.start]
	r.start++
	return ret, nil
}
