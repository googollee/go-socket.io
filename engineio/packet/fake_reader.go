package packet

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/googollee/go-socket.io/engineio/frame"
)

type fakeConnReader struct {
	frames []Frame
}

func NewFakeConnReader(frames []Frame) *fakeConnReader {
	return &fakeConnReader{
		frames: frames,
	}
}

func (r *fakeConnReader) NextReader() (frame.Type, io.ReadCloser, error) {
	if len(r.frames) == 0 {
		return frame.String, nil, io.EOF
	}
	f := r.frames[0]
	r.frames = r.frames[1:]
	return f.FType, ioutil.NopCloser(bytes.NewReader(f.Data)), nil
}

type fakeOneFrameConst struct {
	b byte
}

func (c *fakeOneFrameConst) Read(p []byte) (int, error) {
	p[0] = c.b
	return 1, nil
}

type fakeConstReader struct {
	ft frame.Type
	r  *fakeOneFrameConst
}

func NewFakeConstReader() *fakeConstReader {
	return &fakeConstReader{
		ft: frame.String,
		r: &fakeOneFrameConst{
			b: MESSAGE.StringByte(),
		},
	}
}

func (r *fakeConstReader) NextReader() (frame.Type, io.ReadCloser, error) {
	ft := r.ft
	switch ft {
	case frame.Binary:
		r.ft = frame.String
		r.r.b = MESSAGE.StringByte()
	case frame.String:
		r.ft = frame.Binary
		r.r.b = MESSAGE.BinaryByte()
	}
	return ft, ioutil.NopCloser(r.r), nil
}
