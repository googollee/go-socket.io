package payload

import (
	"encoding/base64"
	"io"
	"io/ioutil"
)

type limitReader struct {
	limit *io.LimitedReader
	b64   io.Reader
}

func newLimitReader(r io.Reader) *limitReader {
	return &limitReader{
		limit: &io.LimitedReader{
			R: r,
		},
	}
}

func (r *limitReader) Limit(n int, b64 bool) {
	r.limit.N = int64(n)
	if b64 {
		r.b64 = base64.NewDecoder(base64.StdEncoding, r.limit)
	} else {
		r.b64 = nil
	}
}

func (r *limitReader) Read(p []byte) (int, error) {
	if r.b64 != nil {
		return r.b64.Read(p)
	}
	return r.limit.Read(p)
}

func (r *limitReader) Close() error {
	if r.limit.N == 0 {
		r.b64 = nil
		return nil
	}
	_, err := io.Copy(ioutil.Discard, r)
	r.b64 = nil
	return err
}
