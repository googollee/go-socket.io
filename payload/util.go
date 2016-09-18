package payload

import "bytes"

func writeBinaryLen(l int64, w *bytes.Buffer) error {
	if l <= 0 {
		if err := w.WriteByte(0x00); err != nil {
			return err
		}
		if err := w.WriteByte(0xff); err != nil {
			return err
		}
		return nil
	}
	max := int64(1)
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

func writeTextLen(l int64, w *bytes.Buffer) error {
	if l <= 0 {
		if err := w.WriteByte('0'); err != nil {
			return err
		}
		if err := w.WriteByte(':'); err != nil {
			return err
		}
		return nil
	}
	max := int64(1)
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

func readBinaryLen(r byteReader) (int64, error) {
	ret := int64(0)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b == 0xff {
			break
		}
		if b > 9 {
			return 0, errInvalidPayload
		}
		ret = ret*10 + int64(b)
	}
	return ret, nil
}

func readTextLen(r byteReader) (int64, error) {
	ret := int64(0)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b == ':' {
			break
		}
		if b < '0' || b > '9' {
			return 0, errInvalidPayload
		}
		ret = ret*10 + int64(b-'0')
	}
	return ret, nil
}
