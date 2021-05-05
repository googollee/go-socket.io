package packet

import (
	"io"

	"github.com/googollee/go-socket.io/engineio/frame"
)

type fakeOneFrameDiscarder struct{}

func (d fakeOneFrameDiscarder) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d fakeOneFrameDiscarder) Close() error {
	return nil
}

type FakeDiscardWriter struct{}

func (w *FakeDiscardWriter) NextWriter(fType frame.Type) (io.WriteCloser, error) {
	return fakeOneFrameDiscarder{}, nil
}
