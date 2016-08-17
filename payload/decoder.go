package payload

import (
	"io"

	"github.com/googollee/go-engine.io/base"
)

type decoder struct {
	conn ConnReader
	lr   *limitReader
	read func() (base.FrameType, base.PacketType, io.Reader, error)
}

func newDecoder(conn ConnReader) *decoder {
	ret := &decoder{
		conn: conn,
		lr:   newLimitReader(conn),
	}
	if conn.SupportBinary() {
		ret.read = ret.binaryRead
	} else {
		ret.read = ret.stringRead
	}
	return ret
}

func (r *decoder) NextReader() (base.FrameType, base.PacketType, io.Reader, error) {
	if err := r.lr.Close(); err != nil {
		return base.FrameInvalid, base.UNKNOWN, nil, err
	}
	return r.read()
}

func (r *decoder) stringRead() (base.FrameType, base.PacketType, io.Reader, error) {
	l, err := readStringLen(r.conn)
	if err != nil {
		return base.FrameInvalid, base.UNKNOWN, nil, err
	}

	ft := base.FrameString
	b, err := r.conn.ReadByte()
	if err != nil {
		return base.FrameInvalid, base.UNKNOWN, nil, err
	}
	l--

	if b == 'b' {
		ft = base.FrameBinary
		b, err = r.conn.ReadByte()
		if err != nil {
			return base.FrameInvalid, base.UNKNOWN, nil, err
		}
		l--
	}

	pt := base.ByteToPacketType(b, base.FrameString)
	r.lr.Limit(l, ft == base.FrameBinary)
	return ft, pt, r.lr, nil
}

func (r *decoder) binaryRead() (base.FrameType, base.PacketType, io.Reader, error) {
	b, err := r.conn.ReadByte()
	if err != nil {
		return base.FrameInvalid, base.UNKNOWN, nil, err
	}
	if b > 1 {
		return base.FrameInvalid, base.UNKNOWN, nil, ErrInvalidPayload
	}
	ft := base.ByteToFrameType(b)

	l, err := readBinaryLen(r.conn)
	if err != nil {
		return base.FrameInvalid, base.UNKNOWN, nil, err
	}

	b, err = r.conn.ReadByte()
	if err != nil {
		return base.FrameInvalid, base.UNKNOWN, nil, err
	}
	pt := base.ByteToPacketType(b, ft)
	l--

	r.lr.Limit(l, false)
	return ft, pt, r.lr, nil
}
