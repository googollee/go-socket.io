package payload

import (
	"bufio"
	"encoding/base64"
	"io"
	"io/ioutil"

	"github.com/googollee/go-engine.io/base"
)

type byteReader interface {
	ReadByte() (byte, error)
	io.Reader
}

type decoder struct {
	supportBinary bool
	rawReader     byteReader

	limitReader io.LimitedReader
	b64Reader   io.Reader
}

func (d *decoder) FeedIn(r io.Reader, supportBinary bool) {
	d.supportBinary = supportBinary
	var ok bool
	d.rawReader, ok = r.(byteReader)
	if !ok {
		d.rawReader = bufio.NewReader(r)
	}
	d.limitReader.R = d.rawReader
}

// If it returns io.EOF, it need feed in new reader.
func (d *decoder) NextReader() (base.FrameType, base.PacketType, io.Reader, error) {
	if d.rawReader == nil {
		return 0, 0, nil, io.EOF
	}
	// discard all remaining data.
	d.Close()

	var read func() (base.FrameType, base.PacketType, error)
	if d.supportBinary {
		read = d.binaryRead
	} else {
		read = d.textRead
	}

	ft, pt, err := read()
	if err != nil {
		return 0, 0, nil, err
	}
	return ft, pt, d, nil
}

func (d *decoder) Read(p []byte) (int, error) {
	if d.b64Reader != nil {
		return d.b64Reader.Read(p)
	}
	return d.limitReader.Read(p)
}

func (d *decoder) Close() error {
	_, err := io.Copy(ioutil.Discard, d)
	return err
}

func (d *decoder) hasFeeded() bool {
	return d.rawReader != nil
}

func (d *decoder) setNextLimit(n int64, b64 bool) {
	d.limitReader.N = n
	if b64 {
		d.b64Reader = base64.NewDecoder(base64.StdEncoding, &d.limitReader)
	} else {
		d.b64Reader = nil
	}
}

func (d *decoder) textRead() (base.FrameType, base.PacketType, error) {
	l, err := readStringLen(d.rawReader)
	if err != nil {
		return 0, 0, err
	}

	ft := base.FrameString
	b, err := d.rawReader.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	l--

	if b == 'b' {
		ft = base.FrameBinary
		b, err = d.rawReader.ReadByte()
		if err != nil {
			return 0, 0, err
		}
		l--
	}

	pt := base.ByteToPacketType(b, base.FrameString)
	d.setNextLimit(l, ft == base.FrameBinary)
	return ft, pt, nil
}

func (d *decoder) binaryRead() (base.FrameType, base.PacketType, error) {
	b, err := d.rawReader.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	if b > 1 {
		return 0, 0, err
	}
	ft := base.ByteToFrameType(b)

	l, err := readBinaryLen(d.rawReader)
	if err != nil {
		return 0, 0, err
	}

	b, err = d.rawReader.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	pt := base.ByteToPacketType(b, ft)
	l--

	d.setNextLimit(l, false)
	return ft, pt, nil
}
