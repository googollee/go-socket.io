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
	io.ReadCloser
	locker    *sync.RWMutex
	closeOnce sync.Once
}

func newReader(r io.ReadCloser, locker *sync.RWMutex) *reader {
	return &reader{
		ReadCloser: r,
		locker:     locker,
	}
}

func (r *reader) Close() error {
	io.Copy(ioutil.Discard, r.ReadCloser)
	err := r.ReadCloser.Close()
	r.closeOnce.Do(func() {
		r.locker.RUnlock()
	})
	return err
}
