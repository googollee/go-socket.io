package payload

import (
	"bufio"
	"io"
	"time"

	"github.com/googollee/go-engine.io/base"
)

type writerArg struct {
	w   io.Writer
	err chan error
}

type encoder struct {
	writerChan    chan io.Writer
	errorChan     chan error
	supportBinary bool
	signal        *Signal
	cache         *frameCache
	deadline      time.Time
}

func newEncoder(supportBinary bool, sig *Signal) *encoder {
	ret := &encoder{
		writerChan:    make(chan io.Writer),
		errorChan:     make(chan error),
		supportBinary: supportBinary,
		signal:        sig,
	}
	ret.cache = newFrameCache(ret)
	return ret
}

func (w *encoder) SetDeadline(t time.Time) error {
	w.deadline = t
	return nil
}

func (w *encoder) NextWriter(ft base.FrameType, pt base.PacketType) (io.WriteCloser, error) {
	b64 := false
	if !w.supportBinary && ft == base.FrameBinary {
		b64 = true
	}
	w.cache.Reset(b64, ft, pt)
	return w.cache, nil
}

func (w *encoder) FlushOut(wr io.Writer) error {
	select {
	case w.writerChan <- wr:
	case <-w.signal.WaitClose():
		return w.signal.LoadError()
	case <-w.signal.WaitPause():
		var err error
		if w.supportBinary {
			_, err = wr.Write([]byte{0x00, 0x01, 0xff, '6'})
		} else {
			_, err = wr.Write([]byte("1:6"))
		}
		return err
	}
	select {
	case err := <-w.errorChan:
		return err
	case <-w.signal.WaitClose():
		return w.signal.LoadError()
	}
}

func (w *encoder) waitWriter() (io.Writer, error) {
	if w.deadline.IsZero() {
		select {
		case arg := <-w.writerChan:
			return arg, nil
		case <-w.signal.WaitClose():
			return nil, w.signal.LoadError()
		}
	}
	select {
	case <-time.After(w.deadline.Sub(time.Now())):
		return nil, w.signal.StoreError(ErrTimeout)
	case arg := <-w.writerChan:
		return arg, nil
	case <-w.signal.WaitClose():
		return nil, w.signal.LoadError()
	}
}

func (w *encoder) closeFrame() error {
	var writeHeader func(ByteWriter) error
	if w.supportBinary {
		writeHeader = w.writeBinaryHeader
	} else {
		if w.cache.ft == base.FrameBinary {
			writeHeader = w.writeB64Header
		} else {
			writeHeader = w.writeStringHeader
		}
	}

	arg, err := w.waitWriter()
	if err != nil {
		return err
	}
	writer, ok := arg.(ByteWriter)
	var flusher *bufio.Writer
	if !ok {
		flusher = bufio.NewWriter(arg)
		writer = flusher
	}

	err = writeHeader(writer)
	if err == nil {
		_, err = writer.Write(w.cache.data.Bytes())
	}
	if err == nil && flusher != nil {
		err = flusher.Flush()
	}
	if err != nil {
		w.signal.StoreError(err)
	}
	select {
	case w.errorChan <- err:
	case <-w.signal.WaitClose():
		return w.signal.LoadError()
	}
	return err
}

func (w *encoder) writeStringHeader(bw ByteWriter) error {
	l := w.cache.data.Len() + 1 // length for packet type
	err := writeStringLen(l, bw)
	if err == nil {
		err = bw.WriteByte(w.cache.pt.StringByte())
	}
	return err
}

func (w *encoder) writeB64Header(bw ByteWriter) error {
	l := w.cache.data.Len() + 2 // length for 'b' and packet type
	err := writeStringLen(l, bw)
	if err == nil {
		err = bw.WriteByte('b')
	}
	if err == nil {
		err = bw.WriteByte(w.cache.pt.StringByte())
	}
	return err
}

func (w *encoder) writeBinaryHeader(bw ByteWriter) error {
	l := w.cache.data.Len() + 1 // length for packet type
	b := w.cache.pt.StringByte()
	if w.cache.ft == base.FrameBinary {
		b = w.cache.pt.BinaryByte()
	}
	err := bw.WriteByte(w.cache.ft.Byte())
	if err == nil {
		err = writeBinaryLen(l, bw)
	}
	if err == nil {
		err = bw.WriteByte(b)
	}
	return err
}
