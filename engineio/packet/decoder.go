package packet

import (
	"io"

	"github.com/googollee/go-socket.io/engineio/frame"
)

// FrameReader is the reader which supports framing.
type FrameReader interface {
	NextReader() (frame.Type, io.ReadCloser, error)
}

type Decoder struct {
	r FrameReader
}

func NewDecoder(r FrameReader) *Decoder {
	return &Decoder{
		r: r,
	}
}

func (e *Decoder) NextReader() (frame.Type, Type, io.ReadCloser, error) {
	ft, r, err := e.r.NextReader()
	if err != nil {
		return 0, 0, nil, err
	}
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		_ = r.Close()
		return 0, 0, nil, err
	}
	return ft, ByteToPacketType(b[0], ft), r, nil
}
