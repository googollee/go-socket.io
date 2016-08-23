package payload

import (
	"encoding/base64"
	"io"
	"io/ioutil"
)

type limitReader struct {
	decoder *decoder
	limit   *io.LimitedReader
	b64     io.Reader
}

func newLimitReader(d *decoder) *limitReader {
	return &limitReader{
		decoder: d,
		limit:   &io.LimitedReader{},
	}
}

func (r *limitReader) SetReader(rd io.Reader, n int, b64 bool) {
	r.limit.R = rd
	r.limit.N = int64(n)
	if b64 {
		r.b64 = base64.NewDecoder(base64.StdEncoding, r.limit)
	} else {
		r.b64 = nil
	}
}

func (r *limitReader) Read(p []byte) (int, error) {
	var read func([]byte) (int, error)
	if r.b64 != nil {
		read = r.b64.Read
	} else {
		read = r.limit.Read
	}
	n, err := read(p)
	if err != nil && err != io.EOF {
		r.decoder.closeFrame(err)
	}
	return n, err
}

func (r *limitReader) Close() error {
	if r.limit.N == 0 {
		r.b64 = nil
		r.decoder.closeFrame(nil)
		return nil
	}
	_, err := io.Copy(ioutil.Discard, r)
	r.b64 = nil
	r.decoder.closeFrame(err)
	return err
}
