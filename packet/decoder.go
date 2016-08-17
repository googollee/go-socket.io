package packet

import (
	"io"

	"github.com/googollee/go-engine.io/base"
)

type decoder struct {
	r FrameReader
}

func newDecoder(r FrameReader) *decoder {
	return &decoder{
		r: r,
	}
}

func (e *decoder) NextReader() (base.FrameType, base.PacketType, io.Reader, error) {
	ft, r, err := e.r.NextReader()
	if err != nil {
		return base.FrameInvalid, base.UNKNOWN, nil, err
	}
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return base.FrameInvalid, base.UNKNOWN, nil, err
	}
	return ft, base.ByteToPacketType(b[0], ft), r, nil
}
