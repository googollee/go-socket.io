package polling

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/transport"
)

const (
	separator    = 0x1e
	binaryPrefix = 'b'
)

func init() {
	transport.Register(transport.Polling, newPolling)
}

type polling struct {
	allocator transport.BufferAllocator
	callbacks transport.Callbacks

	post      chan struct{}
	get       chan struct{}
	closed    chan struct{}
	closeOnce sync.Once

	readBuf  []byte
	writeBuf []byte
	encoder  *encoder
}

func newPolling(pingInterval time.Duration, alloc transport.BufferAllocator, callbacks transport.Callbacks) transport.Transport {
	ret := &polling{
		allocator: alloc,
		callbacks: callbacks,
		post:      make(chan struct{}, 1),
		get:       make(chan struct{}, 1),
		closed:    make(chan struct{}),
	}

	ret.post <- struct{}{}
	ret.get <- struct{}{}
	ret.readBuf = alloc.New()
	ret.writeBuf = alloc.New()
	ret.encoder = newEncoder(pingInterval, ret.closed, ret.writeBuf)

	return ret
}

func (p *polling) Name() string {
	return string(transport.Polling)
}

func (p *polling) Close() error {
	p.closeOnce.Do(func() {
		close(p.closed)

		// wait for all requests.
		<-p.post
		<-p.get

		p.allocator.Free(p.readBuf)
		p.allocator.Free(p.writeBuf)
		p.readBuf = nil
		p.writeBuf = nil
	})

	return nil
}

func (p *polling) SendFrame(ft frame.Type) (io.WriteCloser, error) {
	return p.encoder.NextFrame(ft)
}

func (p *polling) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		p.serveGet(w, r)
	case "POST":
		p.servePost(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid method " + r.Method))
	}
}

type httpError interface {
	error
	Code() int
}

func (p *polling) servePost(w http.ResponseWriter, r *http.Request) {
	select {
	case <-p.post:
		defer func() {
			p.post <- struct{}{}
		}()
	case <-p.closed:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("closed session"))
		return
	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("overlapped post"))
		return
	}

	decoder := newDecoder(p.readBuf, r.Body)
	for {
		ft, rd, err := decoder.NextFrame()
		if err != nil {
			if err == io.EOF {
				break
			}

			p.callbacks.OnError(p, err)
			return
		}

		if err := p.callbacks.OnFrame(p, r, ft, rd); err != nil {
			code := http.StatusInternalServerError
			if he, ok := err.(httpError); ok {
				code = he.Code()
			}
			w.WriteHeader(code)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (p *polling) serveGet(w http.ResponseWriter, r *http.Request) {
	select {
	case <-p.get:
		defer func() {
			p.get <- struct{}{}
		}()
	case <-p.closed:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("closed session"))
		return
	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("overlapped get"))
		return
	}

	for {
		switch err := p.encoder.WriteFramesTo(w); err {
		case ErrPingTimeout:
			p.callbacks.OnPingTimeout(p)
			// loop again to write ping frame out or break if the session is closed.
		case io.EOF:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("closed session"))
			return
		case nil:
			return
		default:
			p.callbacks.OnError(p, err)
			return
		}
	}
}
