package polling

import (
	"io"
	"net/http"
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

type Polling struct {
	allocator transport.BufferAllocator
	callbacks transport.Callbacks

	post   chan struct{}
	get    chan struct{}
	closed chan struct{}

	readBuf  []byte
	writeBuf []byte
	encoder  *encoder
}

func newPolling(pingInterval time.Duration, alloc transport.BufferAllocator, callbacks transport.Callbacks) transport.Transport {
	ret := &Polling{
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

func (p *Polling) Name() string {
	return string(transport.Polling)
}

func (p *Polling) Close() error {
	<-p.post
	<-p.get
	close(p.closed)

	p.encoder.WaitFrameClose()
	p.encoder = nil

	p.allocator.Free(p.readBuf)
	p.allocator.Free(p.writeBuf)
	p.readBuf = nil
	p.writeBuf = nil
	return nil
}

func (p *Polling) SendFrame(ft frame.Type) (io.WriteCloser, error) {
	return p.encoder.NextFrame(ft)
}

func (p *Polling) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		p.serveGet(w, r)
	case "POST":
		p.servePost(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid method " + r.Method))
	}
}

type httpError interface {
	error
	Code() int
}

func (p *Polling) servePost(w http.ResponseWriter, r *http.Request) {
	select {
	case <-p.post:
		defer func() {
			p.post <- struct{}{}
		}()
	case <-p.closed:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("closed session"))
		return
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("overlapped post"))
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
			w.Write([]byte(err.Error()))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (p *Polling) serveGet(w http.ResponseWriter, r *http.Request) {
	select {
	case <-p.get:
		defer func() {
			p.get <- struct{}{}
		}()
	case <-p.closed:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("closed session"))
		return
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("overlapped get"))
		return
	}

	for {
		err := p.encoder.WriteFramesTo(w)
		if err == nil {
			return
		}

		if err == ErrPingTimeout {
			p.callbacks.OnPingTimeout(p)
			continue
		}

		p.callbacks.OnError(p, err)
	}
}
