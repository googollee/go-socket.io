package payload

import (
	"bytes"
	"encoding/base64"
	"io"
	"unicode/utf8"

	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/packet"
)

type writerFeeder interface {
	getWriter() (io.Writer, error)
	putWriter(error) error
}

type encoder struct {
	supportBinary bool
	feeder        writerFeeder

	ft         frame.Type
	pt         packet.Type
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

func (e *encoder) NextWriter(ft frame.Type, pt packet.Type) (io.WriteCloser, error) {
	w, err := e.feeder.getWriter()
	if err != nil {
		return nil, err
	}
	e.rawWriter = w

	e.ft = ft
	e.pt = pt
	e.frameCache.Reset()

	if !e.supportBinary && ft == frame.Binary {
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
	// Need to scan here and add unicode chars to the list
	return e.frameCache.Write(p)
}

func (e *encoder) Close() error {
	if e.b64Writer != nil {
		if err := e.b64Writer.Close(); err != nil {
			return err
		}
	}

	var writeHeader func() error
	if e.supportBinary {
		writeHeader = e.writeBinaryHeader
	} else {
		if e.ft == frame.Binary {
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

	err := writeTextLen(e.calcCodeUnitLength(), &e.header)
	if err == nil {
		err = e.header.WriteByte(e.pt.StringByte())
	}
	return err
}

func (e *encoder) writeB64Header() error {
	l := int64(utf8.RuneCount(e.frameCache.Bytes()) + 2) // length for 'b' and packet type
	err := writeTextLen(l, &e.header)
	if err == nil {
		err = e.header.WriteByte('b')
	}
	if err == nil {
		err = e.header.WriteByte(e.pt.StringByte())
	}
	return err
}

func (e *encoder) calcCodeUnitLength() int64 {
	var l int64 = 1
	var codeUnitSize int64
	bytes := e.frameCache.Bytes()
	for i := range bytes {
		b := bytes[i]
		if b>>3 == 30 {
			// starts with 11110 4 byte unicode char, probably 2 length in JS
			codeUnitSize = 2
		} else if b>>4 == 14 {
			// starts with 1110 3 byte unicode char, probably 1 length in JS
			codeUnitSize = 1
		} else if b>>5 == 6 {
			// starts with 110 2 byte unicode char, , probably 1 length in JS
			codeUnitSize = 1
		} else if b>>6 == 2 {
			// starts with 10 just unicode byte
			codeUnitSize = 0
		} else {
			codeUnitSize = 1
		}
		l = l + codeUnitSize
	}

	return int64(l)
}
func (e *encoder) writeBinaryHeader() error {
	l := int64(e.calcCodeUnitLength()) // length for packet type
	b := e.pt.StringByte()
	if e.ft == frame.Binary {
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
