package payload

import (
	"bytes"
	"encoding/base64"
	"io"

	"github.com/googollee/go-socket.io/connection/base"
)

type writerFeeder interface {
	getWriter() (io.Writer, error)
	putWriter(error) error
}

type encoder struct {
	supportBinary bool
	feeder        writerFeeder

	ft         base.FrameType
	pt         base.PacketType
	header     bytes.Buffer
	frameCache bytes.Buffer
	b64Writer  io.WriteCloser
	rawWriter  io.Writer
}

func (e *encoder) NOOP() []byte {
	if e.supportBinary {
		return []byte{0x00, 0x01, 0xff, '6'}
	}
	return []byte("1:6")
}

func (e *encoder) NextWriter(ft base.FrameType, pt base.PacketType) (io.WriteCloser, error) {
	w, err := e.feeder.getWriter()
	if err != nil {
		return nil, err
	}
	e.rawWriter = w

	e.ft = ft
	e.pt = pt
	e.frameCache.Reset()

	if !e.supportBinary && ft == base.FrameBinary {
		e.b64Writer = base64.NewEncoder(base64.StdEncoding, &e.frameCache)
	} else {
		e.b64Writer = nil
	}
	return e, nil
}

func (e *encoder) Write(p []byte) (int, error) {
	if e.b64Writer != nil {
		return e.b64Writer.Write(p)
	}
	return e.frameCache.Write(p)
}

func (e *encoder) Close() error {
	if e.b64Writer != nil {
		e.b64Writer.Close()
	}

	var writeHeader func() error
	if e.supportBinary {
		writeHeader = e.writeBinaryHeader
	} else {
		if e.ft == base.FrameBinary {
			writeHeader = e.writeB64Header
		} else {
			writeHeader = e.writeTextHeader
		}
	}

	e.header.Reset()
	err := writeHeader()
	if err == nil {
		_, err = e.header.WriteTo(e.rawWriter)
	}
	if err == nil {
		_, err = e.frameCache.WriteTo(e.rawWriter)
	}
	if werr := e.feeder.putWriter(err); werr != nil {
		return werr
	}
	return err
}

func (e *encoder) writeTextHeader() error {
	l := int64(e.frameCache.Len() + 1) // length for packet type
	err := writeTextLen(l, &e.header)
	if err == nil {
		err = e.header.WriteByte(e.pt.StringByte())
	}
	return err
}

func (e *encoder) writeB64Header() error {
	l := int64(e.frameCache.Len() + 2) // length for 'b' and packet type
	err := writeTextLen(l, &e.header)
	if err == nil {
		err = e.header.WriteByte('b')
	}
	if err == nil {
		err = e.header.WriteByte(e.pt.StringByte())
	}
	return err
}

func (e *encoder) writeBinaryHeader() error {
	l := int64(e.frameCache.Len() + 1) // length for packet type
	b := e.pt.StringByte()
	if e.ft == base.FrameBinary {
		b = e.pt.BinaryByte()
	}
	err := e.header.WriteByte(e.ft.Byte())
	if err == nil {
		err = writeBinaryLen(l, &e.header)
	}
	if err == nil {
		err = e.header.WriteByte(b)
	}
	return err
}
