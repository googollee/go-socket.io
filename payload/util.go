package payload

import (
	"io"
	"sync"
)

type byteWriter interface {
	WriteByte(byte) error
}

func writeBinaryLen(l int, w byteWriter) error {
	if l <= 0 {
		if err := w.WriteByte(0x00); err != nil {
			return err
		}
		if err := w.WriteByte(0xff); err != nil {
			return err
		}
		return nil
	}
	max := 1
	for n := l / 10; n > 0; n /= 10 {
		max *= 10
	}
	for max > 0 {
		n := l / max
		if err := w.WriteByte(byte(n)); err != nil {
			return err
		}
		l -= n * max
		max /= 10
	}
	return w.WriteByte(0xff)
}

func writeStringLen(l int, w byteWriter) error {
	if l <= 0 {
		if err := w.WriteByte('0'); err != nil {
			return err
		}
		if err := w.WriteByte(':'); err != nil {
			return err
		}
		return nil
	}
	max := 1
	for n := l / 10; n > 0; n /= 10 {
		max *= 10
	}
	for max > 0 {
		n := l / max
		if err := w.WriteByte(byte(n) + '0'); err != nil {
			return err
		}
		l -= n * max
		max /= 10
	}
	return w.WriteByte(':')
}

func readBinaryLen(r ByteReader) (int, error) {
	ret := 0
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b == 0xff {
			break
		}
		if b > 9 {
			return 0, ErrInvalidPayload
		}
		ret = ret*10 + int(b)
	}
	return ret, nil
}

func readStringLen(r ByteReader) (int, error) {
	ret := 0
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b == ':' {
			break
		}
		if b < '0' || b > '9' {
			return 0, ErrInvalidPayload
		}
		ret = ret*10 + int(b-'0')
	}
	return ret, nil
}

// AtomicError is a error storage.
type AtomicError struct {
	locker sync.RWMutex
	error
}

// Store saves error.
func (e *AtomicError) Store(err error) error {
	e.locker.Lock()
	defer e.locker.Unlock()
	e.error = err
	return err
}

// Load loads error.
func (e *AtomicError) Load() error {
	e.locker.RLock()
	defer e.locker.RUnlock()
	if e.error == nil {
		return io.EOF
	}
	return e.error
}
