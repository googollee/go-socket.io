package engineio

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"sync"
)

const Protocol = 3

type PacketType string

const (
	OPEN    PacketType = "open"
	CLOSE   PacketType = "close"
	PING    PacketType = "ping"
	PONG    PacketType = "pong"
	MESSAGE PacketType = "message"
	UPGRADE PacketType = "upgrade"
	NOOP    PacketType = "noop"
)

func byteToType(b byte) (PacketType, error) {
	if b >= '0' {
		b = b - '0'
	}
	switch b {
	case 0:
		return OPEN, nil
	case 1:
		return CLOSE, nil
	case 2:
		return PING, nil
	case 3:
		return PONG, nil
	case 4:
		return MESSAGE, nil
	case 5:
		return UPGRADE, nil
	case 6:
		return NOOP, nil
	}
	return NOOP, fmt.Errorf("invalid byte 0x%x", b)
}

func (t PacketType) Byte() byte {
	switch t {
	case OPEN:
		return 0
	case CLOSE:
		return 1
	case PING:
		return 2
	case PONG:
		return 3
	case MESSAGE:
		return 4
	case UPGRADE:
		return 5
	}
	return 6
}

type PacketEncoder struct {
	closer io.Closer
	w      io.Writer
}

func NewStringEncoder(w io.Writer, t PacketType) (*PacketEncoder, error) {
	return newEncoder(w, t.Byte()+'0')
}

func NewBinaryEncoder(w io.Writer, t PacketType) (*PacketEncoder, error) {
	return newEncoder(w, t.Byte())
}

func newEncoder(w io.Writer, t byte) (*PacketEncoder, error) {
	if _, err := w.Write([]byte{t}); err != nil {
		return nil, err
	}
	closer, ok := w.(io.Closer)
	if !ok {
		closer = nil
	}
	return &PacketEncoder{
		closer: closer,
		w:      w,
	}, nil
}

type multiCloser struct {
	closers []io.Closer
}

func (c *multiCloser) Append(closer io.Closer) {
	c.closers = append(c.closers, closer)
}

func (c *multiCloser) Close() error {
	for _, closer := range c.closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

func NewB64Encoder(w io.Writer, t PacketType) (*PacketEncoder, error) {
	_, err := w.Write([]byte{'b', t.Byte() + '0'})
	if err != nil {
		return nil, err
	}
	base := base64.NewEncoder(base64.StdEncoding, w)
	closer, ok := w.(io.Closer)
	if !ok {
		closer = base
	} else {
		mc := &multiCloser{}
		mc.Append(base)
		mc.Append(closer)
		closer = mc
	}
	return &PacketEncoder{
		closer: closer,
		w:      base,
	}, nil
}

func (e *PacketEncoder) Write(p []byte) (int, error) {
	return e.w.Write(p)
}

func (e *PacketEncoder) Close() error {
	if e.closer != nil {
		return e.closer.Close()
	}
	return nil
}

type PacketDecoder struct {
	r      io.Reader
	closer io.Closer
	t      PacketType
}

func NewDecoder(r io.Reader) (*PacketDecoder, error) {
	b := []byte{0xff}
	if _, err := r.Read(b); err != nil {
		return nil, err
	}
	var closer io.Closer
	if c, ok := r.(io.Closer); ok {
		closer = c
	}
	if b[0] == 'b' {
		if _, err := r.Read(b); err != nil {
			return nil, err
		}
		r = base64.NewDecoder(base64.StdEncoding, r)
	}
	t, err := byteToType(b[0])
	if err != nil {
		return nil, err
	}
	return &PacketDecoder{
		r:      r,
		closer: closer,
		t:      t,
	}, nil
}

func (d *PacketDecoder) Read(p []byte) (int, error) {
	return d.r.Read(p)
}

func (d *PacketDecoder) Type() PacketType {
	return d.t
}

func (d *PacketDecoder) Close() error {
	if d.closer != nil {
		return d.closer.Close()
	}
	return nil
}

type PayloadEncoder struct {
	buffers  [][]byte
	locker   sync.Mutex
	isString bool
}

func NewStringPayloadEncoder() *PayloadEncoder {
	return &PayloadEncoder{
		isString: true,
	}
}

func NewBinaryPayloadEncoder() *PayloadEncoder {
	return &PayloadEncoder{
		isString: false,
	}
}

type encoder struct {
	*PacketEncoder
	buf          *bytes.Buffer
	binaryPrefix string
	payload      *PayloadEncoder
}

func (e encoder) Close() error {
	if err := e.PacketEncoder.Close(); err != nil {
		return err
	}
	var buffer []byte
	if e.payload.isString {
		buffer = []byte(fmt.Sprintf("%d:%s", e.buf.Len(), e.buf.String()))
	} else {
		buffer = []byte(fmt.Sprintf("%s%d", e.binaryPrefix, e.buf.Len()))
		for i, n := 0, len(buffer); i < n; i++ {
			buffer[i] = buffer[i] - '0'
		}
		buffer = append(buffer, 0xff)
		buffer = append(buffer, e.buf.Bytes()...)
	}

	e.payload.locker.Lock()
	e.payload.buffers = append(e.payload.buffers, buffer)
	e.payload.locker.Unlock()

	return nil
}

func (e *PayloadEncoder) NextString(t PacketType) (io.WriteCloser, error) {
	buf := bytes.NewBuffer(nil)
	pEncoder, err := NewStringEncoder(buf, t)
	if err != nil {
		return nil, err
	}
	return encoder{
		PacketEncoder: pEncoder,
		buf:           buf,
		binaryPrefix:  "0",
		payload:       e,
	}, nil
}

func (e *PayloadEncoder) NextBinary(t PacketType) (io.WriteCloser, error) {
	buf := bytes.NewBuffer(nil)
	var pEncoder *PacketEncoder
	var err error
	if e.isString {
		pEncoder, err = NewB64Encoder(buf, t)
	} else {
		pEncoder, err = NewBinaryEncoder(buf, t)
	}
	if err != nil {
		return nil, err
	}
	return encoder{
		PacketEncoder: pEncoder,
		buf:           buf,
		binaryPrefix:  "1",
		payload:       e,
	}, nil
}

func (e *PayloadEncoder) EncodeTo(w io.Writer) error {
	e.locker.Lock()
	buffers := e.buffers
	e.buffers = nil
	e.locker.Unlock()

	for _, b := range buffers {
		for len(b) > 0 {
			n, err := w.Write(b)
			if err != nil {
				return err
			}
			b = b[n:]
		}
	}
	return nil
}

type PayloadDecoder struct {
	r *bufio.Reader
}

func NewPayloadDecoder(r io.Reader) *PayloadDecoder {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &PayloadDecoder{
		r: br,
	}
}

func (d *PayloadDecoder) Next() (*PacketDecoder, error) {
	firstByte, err := d.r.Peek(1)
	if err != nil {
		return nil, err
	}
	isBinary := firstByte[0] < '0'
	delim := byte(':')
	if isBinary {
		d.r.ReadByte()
		delim = 0xff
	}
	line, err := d.r.ReadBytes(delim)
	if err != nil {
		return nil, err
	}
	l := len(line)
	if l < 1 {
		return nil, fmt.Errorf("invalid input")
	}
	lenByte := line[:l-1]
	if isBinary {
		for i, n := 0, l; i < n; i++ {
			line[i] = line[i] + '0'
		}
	}
	packetLen, err := strconv.ParseInt(string(lenByte), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}
	b := make([]byte, packetLen)
	for buf := b; len(buf) > 0; {
		n, err := d.r.Read(buf)
		if err != nil {
			return nil, err
		}
		buf = buf[n:]
	}
	buf := bytes.NewBuffer(b)
	return NewDecoder(buf)
}
