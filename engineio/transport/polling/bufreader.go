package polling

import (
	"fmt"
	"io"
)

type bufReader struct {
	buf        []byte
	start, end int
	rd         io.Reader
}

func newBufReader(buf []byte, rd io.Reader) *bufReader {
	return &bufReader{
		buf: buf,
		rd:  rd,
	}
}

func (r *bufReader) Fill() error {
	if r.start != r.end {
		return nil
	}

	r.start = 0
	var err error
	r.end, err = r.rd.Read(r.buf)
	return err
}

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

func (r *bufReader) PushBack(n int) error {
	if n > r.start || n < 0 {
		return fmt.Errorf("not enough buf to push back.")
	}

	r.start -= n
	return nil
}

func (r *bufReader) ReadByte() (byte, error) {
	if err := r.Fill(); err != nil {
		return 0, err
	}

	ret := r.buf[r.start]
	r.start++
	return ret, nil
}
