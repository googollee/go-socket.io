package engineio

import (
	"io"
	"sync"
)

type connReader struct {
	*PacketDecoder
	closeChan chan struct{}
}

func newConnReader(d *PacketDecoder, closeChan chan struct{}) *connReader {
	return &connReader{
		PacketDecoder: d,
		closeChan:     closeChan,
	}
}

func (r *connReader) Close() error {
	if r.closeChan == nil {
		return nil
	}
	r.closeChan <- struct{}{}
	r.closeChan = nil
	return nil
}

type connWriter struct {
	io.WriteCloser
	locker *sync.Mutex
}

func newConnWriter(w io.WriteCloser, locker *sync.Mutex) *connWriter {
	return &connWriter{
		WriteCloser: w,
		locker:      locker,
	}
}

func (w *connWriter) Close() error {
	defer func() {
		if w.locker != nil {
			w.locker.Unlock()
			w.locker = nil
		}
	}()
	return w.WriteCloser.Close()
}

type limitReader struct {
	io.Reader
	remain int
}

func newLimitReader(r io.Reader, limit int) *limitReader {
	return &limitReader{
		Reader: r,
		remain: limit,
	}
}

func (r *limitReader) Read(b []byte) (int, error) {
	if r.remain == 0 {
		return 0, io.EOF
	}
	if len(b) > r.remain {
		b = b[:r.remain]
	}
	n, err := r.Reader.Read(b)
	r.remain -= n
	return n, err
}

func (r *limitReader) Close() error {
	if r.remain > 0 {
		b := make([]byte, 10240)
		for {
			_, err := r.Read(b)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}
