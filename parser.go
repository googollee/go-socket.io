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

// PacketType is the type of packet
type packetType string

const (
	_OPEN    packetType = "open"
	_CLOSE   packetType = "close"
	_PING    packetType = "ping"
	_PONG    packetType = "pong"
	_MESSAGE packetType = "message"
	_UPGRADE packetType = "upgrade"
	_NOOP    packetType = "noop"
)

func byteToType(b byte) (packetType, error) {
	switch b {
	case 0:
		return _OPEN, nil
	case 1:
		return _CLOSE, nil
	case 2:
		return _PING, nil
	case 3:
		return _PONG, nil
	case 4:
		return _MESSAGE, nil
	case 5:
		return _UPGRADE, nil
	case 6:
		return _NOOP, nil
	}
	return _NOOP, fmt.Errorf("invalid byte 0x%x", b)
}

// Byte return the byte of type
func (t packetType) Byte() byte {
	switch t {
	case _OPEN:
		return 0
	case _CLOSE:
		return 1
	case _PING:
		return 2
	case _PONG:
		return 3
	case _MESSAGE:
		return 4
	case _UPGRADE:
		return 5
	}
	return 6
}

// packetEncoder is the encoder which encode the packet.
type packetEncoder struct {
	closer io.Closer
	w      io.Writer
}

// NewStringEncoder return the encoder which encode type t to writer w, as string.
func newStringEncoder(w io.Writer, t packetType) (*packetEncoder, error) {
	return newEncoder(w, t.Byte()+'0')
}

// NewBinaryEncoder return the encoder which encode type t to writer w, as binary.
func newBinaryEncoder(w io.Writer, t packetType) (*packetEncoder, error) {
	return newEncoder(w, t.Byte())
}

func newEncoder(w io.Writer, t byte) (*packetEncoder, error) {
	if _, err := w.Write([]byte{t}); err != nil {
		return nil, err
	}
	closer, ok := w.(io.Closer)
	if !ok {
		closer = nil
	}
	return &packetEncoder{
		closer: closer,
		w:      w,
	}, nil
}

// NewB64Encoder return the encoder which encode type t to writer w, as string. When write binary, it uses base64.
func newB64Encoder(w io.Writer, t packetType) (*packetEncoder, error) {
	_, err := w.Write([]byte{'b', t.Byte() + '0'})
	if err != nil {
		return nil, err
	}
	base := base64.NewEncoder(base64.StdEncoding, w)
	return &packetEncoder{
		closer: base,
		w:      base,
	}, nil
}

// Write writes bytes p.
func (e *packetEncoder) Write(p []byte) (int, error) {
	return e.w.Write(p)
}

// Close closes the encoder.
func (e *packetEncoder) Close() error {
	if e.closer != nil {
		return e.closer.Close()
	}
	return nil
}

// packetDecoder is the decoder which decode data to packet.
type packetDecoder struct {
	closer  io.Closer
	r       io.Reader
	t       packetType
	msgType MessageType
}

// NewDecoder return the decoder which decode from reader r.
func newDecoder(r io.Reader) (*packetDecoder, error) {
	var closer io.Closer
	if limit, ok := r.(*limitReader); ok {
		closer = limit
	}
	defer func() {
		if closer != nil {
			closer.Close()
		}
	}()

	b := []byte{0xff}
	if _, err := r.Read(b); err != nil {
		return nil, err
	}
	msgType := MessageText
	if b[0] == 'b' {
		if _, err := r.Read(b); err != nil {
			return nil, err
		}
		r = base64.NewDecoder(base64.StdEncoding, r)
		msgType = MessageBinary
	}
	if b[0] >= '0' {
		b[0] = b[0] - '0'
	} else {
		msgType = MessageBinary
	}
	t, err := byteToType(b[0])
	if err != nil {
		return nil, err
	}
	ret := &packetDecoder{
		closer:  closer,
		r:       r,
		t:       t,
		msgType: msgType,
	}
	closer = nil
	return ret, nil
}

// Read reads packet data to bytes p.
func (d *packetDecoder) Read(p []byte) (int, error) {
	return d.r.Read(p)
}

// Type returns the type of packet.
func (d *packetDecoder) Type() packetType {
	return d.t
}

// MessageType returns the type of message, binary or string.
func (d *packetDecoder) MessageType() MessageType {
	return d.msgType
}

// Close closes the decoder.
func (d *packetDecoder) Close() error {
	if d.closer != nil {
		return d.closer.Close()
	}
	return nil
}

// payloadEncoder is the encoder to encode packets as payload. It can be used in multi-thread.
type payloadEncoder struct {
	buffers  [][]byte
	locker   sync.Mutex
	isString bool
}

// NewStringPayloadEncoder returns the encoder which encode as string.
func newStringPayloadEncoder() *payloadEncoder {
	return &payloadEncoder{
		isString: true,
	}
}

// NewStringPayloadEncoder returns the encoder which encode as binary.
func newBinaryPayloadEncoder() *payloadEncoder {
	return &payloadEncoder{
		isString: false,
	}
}

type encoder struct {
	*packetEncoder
	buf          *bytes.Buffer
	binaryPrefix string
	payload      *payloadEncoder
}

func (e encoder) Close() error {
	if err := e.packetEncoder.Close(); err != nil {
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

// NextString returns the encoder with packet type t and encode as string.
func (e *payloadEncoder) NextString(t packetType) (io.WriteCloser, error) {
	buf := bytes.NewBuffer(nil)
	pEncoder, err := newStringEncoder(buf, t)
	if err != nil {
		return nil, err
	}
	return encoder{
		packetEncoder: pEncoder,
		buf:           buf,
		binaryPrefix:  "0",
		payload:       e,
	}, nil
}

// NextBinary returns the encoder with packet type t and encode as binary.
func (e *payloadEncoder) NextBinary(t packetType) (io.WriteCloser, error) {
	buf := bytes.NewBuffer(nil)
	var pEncoder *packetEncoder
	var err error
	if e.isString {
		pEncoder, err = newB64Encoder(buf, t)
	} else {
		pEncoder, err = newBinaryEncoder(buf, t)
	}
	if err != nil {
		return nil, err
	}
	return encoder{
		packetEncoder: pEncoder,
		buf:           buf,
		binaryPrefix:  "1",
		payload:       e,
	}, nil
}

// EncodeTo writes encoded payload to writer w. It will clear the buffer of encoder.
func (e *payloadEncoder) EncodeTo(w io.Writer) error {
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

// payloadDecoder is the decoder to decode payload.
type payloadDecoder struct {
	r *bufio.Reader
}

// NewPaylaodDecoder returns the payload decoder which read from reader r.
func newPayloadDecoder(r io.Reader) *payloadDecoder {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &payloadDecoder{
		r: br,
	}
}

// Next returns the packet decoder. Make sure it will be closed after used.
func (d *payloadDecoder) Next() (*packetDecoder, error) {
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
	return newDecoder(newLimitReader(d.r, int(packetLen)))
}
