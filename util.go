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

func (w *writer) Close() (err error) {
	w.closeOnce.Do(func() {
		w.locker.Lock()
		defer w.locker.Unlock()
		err = w.WriteCloser.Close()
	})

	return
}

func (w *writer) Write(p []byte) (int, error) {
	w.locker.Lock()
	defer w.locker.Unlock()
	return w.WriteCloser.Write(p)
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

func (r *reader) Close() (err error) {
	r.closeOnce.Do(func() {
		r.locker.RLock()
		io.Copy(ioutil.Discard, r.ReadCloser)
		err = r.ReadCloser.Close()
		r.locker.RUnlock()
	})

	return
}

func (r *reader) Read(p []byte) (n int, err error) {
	r.locker.RLock()
	defer r.locker.RUnlock()
	n, err = r.ReadCloser.Read(p)
	return
}
