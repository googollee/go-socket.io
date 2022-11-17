package payload

import (
	"errors"
	"fmt"
	"io"
)

type delimReader struct {
	delim byte
	r     byteReader
}

func (rd *delimReader) Read(p []byte) (n int, err error) {
	if p == nil {
		return 0, fmt.Errorf("nil bytes")
	}
	b, err := rd.r.ReadBytes(rd.delim)
	switch {
	case errors.Is(err, io.EOF):
		goto done
	case err != nil:
		return 0, err
	}
	if n := len(b); n > 0 && b[n-1] == separator {
		b = b[:n-1]
	}
done:
	return copy(p, b), err
}

func newDelimReader(delim byte, r byteReader) io.Reader {
	return &delimReader{delim: delim, r: r}
}
