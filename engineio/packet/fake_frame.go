package packet

import (
	"bytes"

	"github.com/googollee/go-socket.io/engineio/frame"
)

type fakeFrame struct {
	w    *fakeConnWriter
	typ  frame.Type
	data *bytes.Buffer
}

func newFakeFrame(w *fakeConnWriter, fType frame.Type) *fakeFrame {
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
	w.w.Frames = append(w.w.Frames, Frame{
		FType: w.typ,
		Data:  w.data.Bytes(),
	})
	return nil
}
