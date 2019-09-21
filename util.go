package engineio

import (
	"io"
	"io/ioutil"
	"sync"
)

type writer struct {
	io.WriteCloser
	locker    sync.Locker
	closeOnce sync.Once
}

func newWriter(w io.WriteCloser, locker sync.Locker) *writer {
	return &writer{
		WriteCloser: w,
		locker:      locker,
	}
}

func (w *writer) Close() (err error) {
	w.closeOnce.Do(func() {
		w.locker.Lock()
		defer w.locker.Unlock()
		err = w.Close()
	})

	return
}

func (w *writer) Write(p []byte) (int, error) {
	w.locker.Lock()
	defer w.locker.Unlock()
	return w.Write(p)
}

type reader struct {
	io.ReadCloser
	locker    sync.Locker
	closeOnce sync.Once
}

func newReader(r io.ReadCloser, locker sync.Locker) *reader {
	return &reader{
		ReadCloser: r,
		locker:     locker,
	}
}

func (r *reader) Close() (err error) {
	r.closeOnce.Do(func() {
		r.locker.Lock()
		io.Copy(ioutil.Discard, r)
		err = r.Close()
		r.locker.Unlock()
	})

	return
}

func (r *reader) Read(p []byte) (n int, err error) {
	r.locker.Lock()
	defer r.locker.Unlock()
	n, err = r.Read(p)
	return
}
