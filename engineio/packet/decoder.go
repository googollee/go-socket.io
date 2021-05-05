package packet

import (
	"io"
)

type decoder struct {
	r FrameReader
}

func newDecoder(r FrameReader) *decoder {
	return &decoder{
		r: r,
	}
}

func (e *decoder) NextReader() (FrameType, PacketType, io.ReadCloser, error) {
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
