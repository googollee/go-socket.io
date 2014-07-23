package socketio

import (
	"io"
)

type WriterHelper struct {
	writer io.Writer
	err    error
}

func NewWriterHelper(w io.Writer) *WriterHelper {
	return &WriterHelper{
		writer: w,
	}
}

func (h *WriterHelper) Write(p []byte) {
	if h.err != nil {
		return
	}
	for len(p) > 0 {
		n, err := h.writer.Write(p)
		if err != nil {
			h.err = err
			return
		}
		p = p[n:]
	}
}

func (h *WriterHelper) Error() error {
	return h.err
}
