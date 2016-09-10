package engineio

import (
	"io"
	"io/ioutil"
	"sync"
)

type writer struct {
	io.WriteCloser
	locker    *sync.RWMutex
	closeOnce sync.Once
}

func newWriter(w io.WriteCloser, locker *sync.RWMutex) *writer {
	return &writer{
		WriteCloser: w,
		locker:      locker,
	}
}

func (w *writer) Close() error {
	err := w.WriteCloser.Close()
	w.closeOnce.Do(func() {
		w.locker.RUnlock()
	})
	return err
}

type reader struct {
	io.Reader
	locker    *sync.RWMutex
	closeOnce sync.Once
}

func newReader(r io.Reader, locker *sync.RWMutex) *reader {
	return &reader{
		Reader: r,
		locker: locker,
	}
}

func (r *reader) Close() (err error) {
	r.closeOnce.Do(func() {
		_, err = io.Copy(ioutil.Discard, r.Reader)
		r.locker.RUnlock()
	})
	return
}
