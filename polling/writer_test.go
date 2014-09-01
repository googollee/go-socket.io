package polling

import (
	"bytes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWriter(t *testing.T) {

	Convey("Wait close", t, func() {
		w := newFakeWriteCloser()
		sendChan := MakeSendChan()

		select {
		case <-sendChan:
			panic("should not run here")
		default:
		}

		writer := NewWriter(w, sendChan)
		err := writer.Close()
		So(err, ShouldBeNil)

		select {
		case <-sendChan:
		default:
			panic("should not run here")
		}

		select {
		case <-sendChan:
			panic("should not run here")
		default:
		}
	})

	Convey("Many writer with close", t, func() {
		sendChan := MakeSendChan()

		for i := 0; i < 10; i++ {
			w := newFakeWriteCloser()
			writer := NewWriter(w, sendChan)
			err := writer.Close()
			So(err, ShouldBeNil)
		}

		select {
		case <-sendChan:
		default:
			panic("should not run here")
		}

		select {
		case <-sendChan:
			panic("should not run here")
		default:
		}
	})

}

type fakeWriteCloser struct {
	*bytes.Buffer
}

func newFakeWriteCloser() *fakeWriteCloser {
	return &fakeWriteCloser{
		Buffer: bytes.NewBuffer(nil),
	}
}

func (f *fakeWriteCloser) Close() error {
	return nil
}
