package packet

import (
	"bytes"
	"io"
	"io/ioutil"
)

type fakeConnReader struct {
	frames []Frame
}

func newFakeConnReader(frames []Frame) *fakeConnReader {
	return &fakeConnReader{
		frames: frames,
	}
}

func (r *fakeConnReader) NextReader() (FrameType, io.ReadCloser, error) {
	if len(r.frames) == 0 {
		return FrameString, nil, io.EOF
	}
	f := r.frames[0]
	r.frames = r.frames[1:]
	return f.typ, ioutil.NopCloser(bytes.NewReader(f.data)), nil
}

type fakeFrame struct {
	w    *fakeConnWriter
	typ  FrameType
	data *bytes.Buffer
}

func newFakeFrame(w *fakeConnWriter, fType FrameType) *fakeFrame {
	return &fakeFrame{
		w:    w,
		typ:  fType,
		data: bytes.NewBuffer(nil),
	}
}

func (w *fakeFrame) Write(p []byte) (int, error) {
	return w.data.Write(p)
}

func (w *fakeFrame) Read(p []byte) (int, error) {
	return w.data.Read(p)
}

func (w *fakeFrame) Close() error {
	if w.w == nil {
		return nil
	}
	w.w.frames = append(w.w.frames, Frame{
		typ:  w.typ,
		data: w.data.Bytes(),
	})
	return nil
}

type fakeConnWriter struct {
	frames []Frame
}

func newFakeConnWriter() *fakeConnWriter {
	return &fakeConnWriter{}
}

func (w *fakeConnWriter) NextWriter(fType FrameType) (io.WriteCloser, error) {
	return newFakeFrame(w, fType), nil
}

type fakeOneFrameConst struct {
	b byte
}

func (c *fakeOneFrameConst) Read(p []byte) (int, error) {
	p[0] = c.b
	return 1, nil
}

type fakeConstReader struct {
	ft FrameType
	r  *fakeOneFrameConst
}

func newFakeConstReader() *fakeConstReader {
	return &fakeConstReader{
		ft: FrameString,
		r: &fakeOneFrameConst{
			b: MESSAGE.StringByte(),
		},
	}
}

func (r *fakeConstReader) NextReader() (FrameType, io.ReadCloser, error) {
	ft := r.ft
	switch ft {
	case FrameBinary:
		r.ft = FrameString
		r.r.b = MESSAGE.StringByte()
	case FrameString:
		r.ft = FrameBinary
		r.r.b = MESSAGE.BinaryByte()
	}
	return ft, ioutil.NopCloser(r.r), nil
}

type fakeOneFrameDiscarder struct{}

func (d fakeOneFrameDiscarder) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d fakeOneFrameDiscarder) Close() error {
	return nil
}

type fakeDiscardWriter struct{}

func (w *fakeDiscardWriter) NextWriter(fType FrameType) (io.WriteCloser, error) {
	return fakeOneFrameDiscarder{}, nil
}
