package payload

import (
	"bufio"
	"encoding/base64"
	"io"

	"github.com/vchitai/go-socket.io/v4/engineio/frame"
	"github.com/vchitai/go-socket.io/v4/engineio/packet"
)

type byteReader interface {
	ReadByte() (byte, error)
	ReadBytes(delim byte) ([]byte, error)
	io.Reader
}

type readerFeeder interface {
	getReader() (io.Reader, bool, error)
	putReader(error) error
}

type decoder struct {
	b64Reader   io.Reader
	limitReader io.Reader
	rawReader   byteReader
	feeder      readerFeeder

	ft            frame.Type
	pt            packet.Type
	supportBinary bool
}

func (d *decoder) NextReader() (frame.Type, packet.Type, io.ReadCloser, error) {
	if d.rawReader == nil {
		r, supportBinary, err := d.feeder.getReader()
		if err != nil {
			return 0, 0, nil, err
		}
		br, ok := r.(byteReader)
		if !ok {
			br = bufio.NewReader(r)
		}
		if err := d.setNextReader(br, supportBinary); err != nil {
			return 0, 0, nil, d.sendError(err)
		}
	}

	return d.ft, d.pt, d, nil
}

func (d *decoder) Read(p []byte) (int, error) {
	if d.b64Reader != nil {
		return d.b64Reader.Read(p)
	}
	return d.limitReader.Read(p)
}

func (d *decoder) Close() error {
	if _, err := io.Copy(io.Discard, d); err != nil {
		return d.sendError(err)
	}
	err := d.setNextReader(d.rawReader, d.supportBinary)
	if err != nil {
		if err != io.EOF {
			return d.sendError(err)
		}
		d.rawReader = nil
		d.limitReader = nil
		d.b64Reader = nil
		err = d.sendError(nil)
	}
	return err
}

func (d *decoder) setNextReader(r byteReader, supportBinary bool) error {
	var read func(byteReader) (frame.Type, packet.Type, error)
	if supportBinary {
		read = d.binaryRead
	} else {
		read = d.textRead
	}

	ft, pt, err := read(r)
	if err != nil {
		return err
	}

	d.ft = ft
	d.pt = pt
	d.rawReader = r
	d.limitReader = newDelimReader(separator, r)
	d.supportBinary = supportBinary
	if ft == frame.Binary {
		d.b64Reader = base64.NewDecoder(base64.StdEncoding, d.limitReader)
	} else {
		d.b64Reader = nil
	}
	return nil
}

func (d *decoder) sendError(err error) error {
	if e := d.feeder.putReader(err); e != nil {
		return e
	}
	return err
}

func (d *decoder) textRead(r byteReader) (frame.Type, packet.Type, error) {
	ft := frame.String
	b, err := r.ReadByte()
	if err != nil {
		return 0, 0, err
	}

	if b == 'b' {
		ft = frame.Binary
		b, err = r.ReadByte()
		if err != nil {
			return 0, 0, err
		}
	}

	pt := packet.ByteToPacketType(b, frame.String)
	return ft, pt, nil
}

func (d *decoder) binaryRead(r byteReader) (frame.Type, packet.Type, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	if b > 1 {
		return 0, 0, errInvalidPayload
	}
	ft := frame.ByteToFrameType(b)

	b, err = r.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	pt := packet.ByteToPacketType(b, ft)

	return ft, pt, nil
}
