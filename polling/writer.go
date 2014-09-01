package polling

import (
	"io"
)

func MakeSendChan() chan bool {
	return make(chan bool, 1)
}

type Writer struct {
	io.WriteCloser
	sendChan chan bool
}

func NewWriter(w io.WriteCloser, sendChan chan bool) *Writer {
	return &Writer{
		WriteCloser: w,
		sendChan:    sendChan,
	}
}

func (w *Writer) Close() error {
	select {
	case w.sendChan <- true:
	default:
	}
	return w.WriteCloser.Close()
}
