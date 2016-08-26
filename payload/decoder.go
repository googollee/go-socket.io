package payload

import (
	"bufio"
	"io"
	"sync/atomic"
	"time"

	"github.com/googollee/go-engine.io/base"
)

type readerArg struct {
	r   io.Reader
	typ base.FrameType
}

type decoder struct {
	errorChan   chan error
	readerChan  chan readerArg
	lastType    base.FrameType
	lastReader  ByteReader
	limitReader *limitReader
	closed      chan struct{}
	err         *atomic.Value
	deadline    time.Time
}

func newDecoder(closed chan struct{}, err *atomic.Value) *decoder {
	ret := &decoder{
		errorChan:  make(chan error),
		readerChan: make(chan readerArg),
		closed:     closed,
		err:        err,
	}
	ret.limitReader = newLimitReader(ret)
	return ret
}

func (r *decoder) SetDeadline(t time.Time) error {
	r.deadline = t
	return nil
}

func (r *decoder) NextReader() (base.FrameType, base.PacketType, io.Reader, error) {
	if r.lastReader == nil {
		arg, err := r.waitReader()
		if err != nil {
			return r.returnError(err)
		}

		br, ok := arg.r.(ByteReader)
		if !ok {
			br = bufio.NewReader(arg.r)
		}
		r.lastReader = br
		r.lastType = arg.typ
	} else {
		r.limitReader.Close()
	}

	for {
		var read func(br ByteReader) (base.FrameType, base.PacketType, io.Reader, error)
		switch r.lastType {
		case base.FrameBinary:
			read = r.binaryRead
		case base.FrameString:
			read = r.stringRead
		default:
			return r.returnError(ErrInvalidPayload)
		}

		ft, pt, ret, err := read(r.lastReader)
		if err != io.EOF {
			if err != nil {
				r.closeFrame(err)
				return r.returnError(err)
			}
			return ft, pt, ret, nil
		}
		r.closeFrame(nil)

		arg, err := r.waitReader()
		if err != nil {
			return r.returnError(err)
		}

		br, ok := arg.r.(ByteReader)
		if !ok {
			br = bufio.NewReader(arg.r)
		}
		r.lastReader = br
		r.lastType = arg.typ
	}
}

func (r *decoder) FeedIn(typ base.FrameType, rd io.Reader) error {
	select {
	case r.readerChan <- readerArg{
		r:   rd,
		typ: typ,
	}:
	case <-r.closed:
		return r.err.Load().(error)
	}
	select {
	case err := <-r.errorChan:
		return err
	case <-r.closed:
		return r.err.Load().(error)
	}
}

func (r *decoder) waitReader() (readerArg, error) {
	if r.deadline.IsZero() {
		select {
		case ret := <-r.readerChan:
			return ret, nil
		case <-r.closed:
			return readerArg{}, r.err.Load().(error)
		}
	}

	waiting := r.deadline.Sub(time.Now())
	select {
	case <-time.After(waiting):
		return readerArg{}, ErrTimeout
	case ret := <-r.readerChan:
		return ret, nil
	case <-r.closed:
		return readerArg{}, r.err.Load().(error)
	}
}

func (r *decoder) closeFrame(err error) {
	select {
	case r.errorChan <- err:
	case <-r.closed:
	}
}

func (r *decoder) stringRead(br ByteReader) (base.FrameType, base.PacketType, io.Reader, error) {
	l, err := readStringLen(br)
	if err != nil {
		return 0, 0, nil, err
	}

	ft := base.FrameString
	b, err := br.ReadByte()
	if err != nil {
		return r.returnError(err)
	}
	l--

	if b == 'b' {
		ft = base.FrameBinary
		b, err = br.ReadByte()
		if err != nil {
			return r.returnError(err)
		}
		l--
	}

	pt := base.ByteToPacketType(b, base.FrameString)
	r.limitReader.SetReader(br, l, ft == base.FrameBinary)
	return ft, pt, r.limitReader, nil
}

func (r *decoder) binaryRead(br ByteReader) (base.FrameType, base.PacketType, io.Reader, error) {
	b, err := br.ReadByte()
	if err != nil {
		return 0, 0, nil, err
	}
	if b > 1 {
		return r.returnError(err)
	}
	ft := base.ByteToFrameType(b)

	l, err := readBinaryLen(br)
	if err != nil {
		return r.returnError(err)
	}

	b, err = br.ReadByte()
	if err != nil {
		return r.returnError(err)
	}
	pt := base.ByteToPacketType(b, ft)
	l--

	r.limitReader.SetReader(br, l, false)
	return ft, pt, r.limitReader, nil
}

func (r *decoder) returnError(err error) (base.FrameType, base.PacketType, io.Reader, error) {
	r.err.Store(err)
	return 0, 0, nil, err
}
