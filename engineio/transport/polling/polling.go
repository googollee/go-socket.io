package polling

import (
	"errors"
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

var (
	ErrNoEnoughBuf          = errors.New("not enough buf to push back")
	ErrNoSpace              = errors.New("no enough space to write")
	ErrPingTimeout          = errors.New("ping timeout")
	ErrSeparatorInTextFrame = errors.New("should not write 0x1e to text frames")
	ErrNonCloseFrame        = errors.New("has a non-closed frame")
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

func (p *polling) PrepareHTTP(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (p *polling) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		p.serveGet(w, r)
	case "POST":
		p.servePost(w, r)
	default:
		p.responseHTTP(w, r, http.StatusBadRequest, "invalid method "+r.Method)
	}
}

func (p *polling) servePost(w http.ResponseWriter, r *http.Request) {
	select {
	case <-p.post:
		defer func() {
			p.post <- struct{}{}
		}()
	case <-p.closed:
		p.responseHTTP(w, r, http.StatusBadRequest, "closed session")
		return
	default:
		p.responseHTTP(w, r, http.StatusBadRequest, "overlapped post")
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
			if he, ok := err.(transport.HTTPError); ok {
				code = he.Code()
			}
			p.responseHTTP(w, r, code, err.Error())
			return
		}
	}

	p.responseHTTP(w, r, http.StatusOK, "ok")
}

func (p *polling) serveGet(w http.ResponseWriter, r *http.Request) {
	select {
	case <-p.get:
		defer func() {
			p.get <- struct{}{}
		}()
	case <-p.closed:
		p.responseHTTP(w, r, http.StatusBadRequest, "closed session")
		return
	default:
		p.responseHTTP(w, r, http.StatusBadRequest, "overlapped get")
		return
	}

	for {
		switch err := p.encoder.WriteFramesTo(w); err {
		case ErrPingTimeout:
			p.callbacks.OnPingTimeout(p)
			// loop again to write ping frame out or break if the session is closed.
		case io.EOF:
			p.responseHTTP(w, r, http.StatusBadRequest, "closed session")
			return
		case nil:
			return
		default:
			p.callbacks.OnError(p, err)
			return
		}
	}
}

func (p *polling) responseHTTP(w http.ResponseWriter, r *http.Request, code int, msg string) {
	w.WriteHeader(code)
	data := []byte(msg)
	for len(data) > 0 {
		n, err := w.Write([]byte(msg))
		data = data[n:]
		if err != nil {
			p.callbacks.OnError(p, err)
			return
		}
	}
}
