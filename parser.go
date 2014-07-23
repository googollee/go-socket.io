package socketio

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/googollee/go-engine.io"
	"io"
	"io/ioutil"
	"strconv"
)

const Protocol = 4

type PacketType int

const (
	CONNECT PacketType = iota
	DISCONNECT
	EVENT
	ACK
	ERROR
	BINARY_EVENT
	BINARY_ACK
)

func (t PacketType) String() string {
	switch t {
	case CONNECT:
		return "connect"
	case DISCONNECT:
		return "disconnect"
	case EVENT:
		return "event"
	case ACK:
		return "ack"
	case ERROR:
		return "error"
	case BINARY_EVENT:
		return "binary_event"
	case BINARY_ACK:
		return "binary_ack"
	}
	return fmt.Sprintf("unknown(%d)", t)
}

type FrameReader interface {
	NextReader() (engineio.MessageType, io.ReadCloser, error)
}

type FrameWriter interface {
	NextWriter(engineio.MessageType) (io.WriteCloser, error)
}

type Packet struct {
	Type         PacketType
	NSP          string
	Id           int
	Data         interface{}
	attachNumber int
}

type Encoder struct {
	w   FrameWriter
	err error
}

func NewEncoder(w FrameWriter) *Encoder {
	return &Encoder{
		w: w,
	}
}

func (e *Encoder) Encode(v Packet) error {
	attachments := encodeAttachments(v.Data)
	v.attachNumber = len(attachments)
	if v.attachNumber > 0 {
		v.Type += BINARY_EVENT - EVENT
	}
	if err := e.encodePacket(v); err != nil {
		return err
	}
	for _, a := range attachments {
		if err := e.writeBinary(a); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodePacket(v Packet) error {
	writer, err := e.w.NextWriter(engineio.MessageText)
	if err != nil {
		return err
	}
	defer writer.Close()

	w := NewTrimWriter(writer, "\n")
	wh := NewWriterHelper(w)
	wh.Write([]byte{byte(v.Type) + '0'})
	if v.Type == BINARY_EVENT || v.Type == BINARY_ACK {
		wh.Write([]byte(fmt.Sprintf("%d-", v.attachNumber)))
	}
	needEnd := false
	if v.NSP != "" {
		wh.Write([]byte(v.NSP))
		needEnd = true
	}
	if v.Id >= 0 {
		f := "%d"
		if needEnd {
			f = ",%d"
			needEnd = false
		}
		wh.Write([]byte(fmt.Sprintf(f, v.Id)))
	}
	if v.Data != nil {
		if needEnd {
			wh.Write([]byte{','})
			needEnd = false
		}
		if wh.Error() != nil {
			return wh.Error()
		}
		encoder := json.NewEncoder(w)
		return encoder.Encode(v.Data)
	}
	return wh.Error()
}

func (e *Encoder) writeBinary(r io.Reader) error {
	writer, err := e.w.NextWriter(engineio.MessageBinary)
	if err != nil {
		return err
	}
	defer writer.Close()

	if _, err := io.Copy(writer, r); err != nil {
		return err
	}
	return nil

}

type Decoder struct {
	reader        FrameReader
	message       string
	current       io.Reader
	currentCloser io.Closer
}

func NewDecoder(r FrameReader) *Decoder {
	return &Decoder{
		reader: r,
	}
}

func (d *Decoder) Decode(v *Packet) error {
	ty, r, err := d.reader.NextReader()
	if err != nil {
		return err
	}
	if d.current != nil {
		d.currentCloser.Close()
		d.current = nil
		d.currentCloser = nil
	}
	defer func() {
		if d.current == nil {
			r.Close()
		}
	}()

	if ty != engineio.MessageText {
		return fmt.Errorf("need text package")
	}
	reader := bufio.NewReader(r)

	v.Id = -1

	t, err := reader.ReadByte()
	if err != nil {
		return err
	}
	v.Type = PacketType(t - '0')

	if v.Type == BINARY_EVENT || v.Type == BINARY_ACK {
		num, err := reader.ReadBytes('-')
		if err != nil {
			return err
		}
		numLen := len(num)
		if numLen == 0 {
			return fmt.Errorf("invalid packet")
		}
		n, err := strconv.ParseInt(string(num[:numLen-1]), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid packet")
		}
		v.attachNumber = int(n)
	}

	next, err := reader.Peek(1)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	if len(next) == 0 {
		return fmt.Errorf("invalid packet")
	}

	if next[0] == '/' {
		path, err := reader.ReadBytes(',')
		if err != nil && err != io.EOF {
			return err
		}
		pathLen := len(path)
		if pathLen == 0 {
			return fmt.Errorf("invalid packet")
		}
		if err == nil {
			path = path[:pathLen-1]
		}
		v.NSP = string(path)
		if err == io.EOF {
			return nil
		}
	}

	id := bytes.NewBuffer(nil)
	finish := false
	for {
		next, err := reader.Peek(1)
		if err == io.EOF {
			finish = true
			break
		}
		if err != nil {
			return err
		}
		if '0' <= next[0] && next[0] <= '9' {
			if err := id.WriteByte(next[0]); err != nil {
				return err
			}
		} else {
			break
		}
		reader.ReadByte()
	}
	if id.Len() > 0 {
		id, err := strconv.ParseInt(id.String(), 10, 64)
		if err != nil {
			return err
		}
		v.Id = int(id)
	}
	if finish {
		return nil
	}

	switch v.Type {
	case EVENT:
		fallthrough
	case BINARY_EVENT:
		msgReader, err := newMessageReader(reader)
		if err != nil {
			return err
		}
		d.message = msgReader.Message()
		d.current = msgReader
		d.currentCloser = r
	case ACK:
		fallthrough
	case BINARY_ACK:
		d.current = reader
		d.currentCloser = r
	}
	return nil
}

func (d *Decoder) Message() string {
	return d.message
}

func (d *Decoder) DecodeData(v *Packet) error {
	if d.current == nil {
		return nil
	}
	defer func() {
		d.currentCloser.Close()
		d.current = nil
		d.currentCloser = nil
	}()
	decoder := json.NewDecoder(d.current)
	if err := decoder.Decode(v.Data); err != nil {
		return err
	}
	if v.Type == BINARY_EVENT || v.Type == BINARY_ACK {
		binary, err := d.decodeBinary(v.attachNumber)
		if err != nil {
			return err
		}
		if err := decodeAttachments(v.Data, binary); err != nil {
			return err
		}
		v.Type -= BINARY_EVENT - EVENT
	}
	return nil
}

func (d *Decoder) decodeBinary(num int) ([][]byte, error) {
	ret := make([][]byte, num)
	for i := 0; i < num; i++ {
		t, r, err := d.reader.NextReader()
		if err != nil {
			return nil, err
		}
		if t == engineio.MessageText {
			return nil, fmt.Errorf("need binary")
		}
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		ret[i] = b
	}
	return ret, nil
}
