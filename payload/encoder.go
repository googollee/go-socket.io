package payload

import (
	"bytes"
	"io"

	"github.com/googollee/go-engine.io/base"
)

type encoder struct {
	conn   ConnWriter
	header *bytes.Buffer
	cache  *frameCache
}

func newEncoder(conn ConnWriter) *encoder {
	ret := &encoder{
		conn:   conn,
		header: bytes.NewBuffer(nil),
	}
	ret.cache = newFrameCache(ret)
	return ret
}

func (w *encoder) NextWriter(ft base.FrameType, pt base.PacketType) (io.WriteCloser, error) {
	b64 := false
	if !w.conn.SupportBinary() && ft == base.FrameBinary {
		b64 = true
	}
	w.cache.Reset(b64, ft, pt)
	return w.cache, nil
}

func (w *encoder) closeFrame() error {
	var writeHeader func() error
	if w.conn.SupportBinary() {
		writeHeader = w.writeBinaryHeader
	} else {
		if w.cache.ft == base.FrameBinary {
			writeHeader = w.writeB64Header
		} else {
			writeHeader = w.writeStringHeader
		}
	}

	w.header.Reset()
	if err := writeHeader(); err != nil {
		return err
	}
	return w.conn.WriteFrame(w.header.Bytes(), w.cache.data.Bytes())
}

func (w *encoder) writeStringHeader() error {
	l := w.cache.data.Len() + 1 // length for packet type
	if err := writeStringLen(l, w.header); err != nil {
		return err
	}
	if err := w.header.WriteByte(w.cache.pt.StringByte()); err != nil {
		return err
	}
	return nil
}

func (w *encoder) writeB64Header() error {
	l := w.cache.data.Len() + 2 // length for 'b' and packet type
	if err := writeStringLen(l, w.header); err != nil {
		return err
	}
	if err := w.header.WriteByte('b'); err != nil {
		return err
	}
	if err := w.header.WriteByte(w.cache.pt.StringByte()); err != nil {
		return err
	}
	return nil
}

func (w *encoder) writeBinaryHeader() error {
	if err := w.header.WriteByte(w.cache.ft.Byte()); err != nil {
		return err
	}
	l := w.cache.data.Len() + 1 // length for packet type
	if err := writeBinaryLen(l, w.header); err != nil {
		return err
	}
	var err error
	switch w.cache.ft {
	case base.FrameString:
		err = w.header.WriteByte(w.cache.pt.StringByte())
	case base.FrameBinary:
		err = w.header.WriteByte(w.cache.pt.BinaryByte())
	}
	if err != nil {
		return err
	}
	return nil
}
