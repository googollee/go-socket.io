package payload

import (
	"fmt"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googollee/go-engine.io/base"
)

type readArg struct {
	r             io.Reader
	supportBinary bool
}

// Payload does encode or decode to payload protocol.
type Payload struct {
	close     chan struct{}
	closeOnce sync.Once
	err       atomic.Value

	pause        chan struct{}
	pauseLocker  sync.RWMutex
	pauseWaiter  sync.WaitGroup
	waiterLocker sync.RWMutex

	readerChan   chan readArg
	reading      int64
	readError    chan error
	readDeadline atomic.Value
	decoder      decoder

	writerChan    chan io.Writer
	writeError    chan error
	writeDeadline atomic.Value
	encoder       encoder
}

// New returns a new payload.
func New(supportBinary bool) *Payload {
	ret := &Payload{
		close:      make(chan struct{}),
		pause:      make(chan struct{}),
		readerChan: make(chan readArg),
		readError:  make(chan error),
		writerChan: make(chan io.Writer),
		writeError: make(chan error),
	}
	ret.readDeadline.Store(time.Time{})
	ret.writeDeadline.Store(time.Time{})
	ret.encoder.supportBinary = supportBinary
	ret.encoder.encoderWriter = ret
	return ret
}

// FeedIn feeds in a new reader for NextReader.
// Multi-FeedIn needs be called sync.
//
// If Close called when FeedIn, it returns io.EOF.
// If Pause called when FeedIn, it returns ErrPaused.
// If NextReader has timeout, it returns ErrTimeout.
// If read error while FeedIn, it returns read error.
func (p *Payload) FeedIn(r io.Reader, supportBinary bool) error {
	select {
	case <-p.close:
		return p.load()
	default:
	}

	select {
	case <-p.close:
		return p.load()
	case <-p.pauseChan():
		return newOpError("payload", errPaused)
	default:
	}

	for {
		after, ok := p.readTimeout()
		if !ok {
			return p.store("read", errTimeout)
		}
		p.waiterLocker.RLock()
		select {
		case <-p.close:
			p.waiterLocker.RUnlock()
			return p.load()
		case <-after:
			// it may changed during wait, need check again
			p.waiterLocker.RUnlock()
			continue
		case <-p.pause:
			p.waiterLocker.RUnlock()
			return newOpError("payload", errPaused)
		case p.readerChan <- readArg{
			r:             r,
			supportBinary: supportBinary,
		}:
			p.pauseWaiter.Add(1)
			p.waiterLocker.RUnlock()
		}
		break
	}
	defer p.pauseWaiter.Done()

	for {
		after, ok := p.readTimeout()
		if !ok {
			return p.store("read", errTimeout)
		}
		select {
		case <-after:
			// it may changed during wait, need check again
			continue
		case err := <-p.readError:
			return p.store("read", err)
		}
	}
}

// FlushOut write data from NextWriter.
// FlushOut needs be called sync.
//
// If Close called when Flushout,  it return io.EOF.
// If Pause called when Flushout, it flushs out a NOOP message and return
// nil.
// If NextWriter has timeout, it returns ErrTimeout.
// If write error while FlushOut, it returns write error.
func (p *Payload) FlushOut(w io.Writer) error {
	select {
	case <-p.close:
		return p.load()
	default:
	}

	select {
	case <-p.close:
		return p.load()
	case <-p.pauseChan():
		return newOpError("payload", errPaused)
	default:
	}

	for {
		after, ok := p.writeTimeout()
		if !ok {
			return p.store("write", errTimeout)
		}
		p.waiterLocker.RLock()
		select {
		case <-p.close:
			p.waiterLocker.RUnlock()
			return p.load()
		case <-p.pause:
			p.waiterLocker.RUnlock()
			return newOpError("payload", errPaused)
		case <-after:
			p.waiterLocker.RUnlock()
			continue
		case p.writerChan <- w:
			p.pauseWaiter.Add(1)
			p.waiterLocker.RUnlock()
		}
		break
	}
	defer p.pauseWaiter.Done()

	for {
		after, ok := p.writeTimeout()
		if !ok {
			return p.store("write", errTimeout)
		}
		select {
		case <-after:
			// it may changed during wait, need check again
		case err := <-p.writeError:
			return p.store("write", err)
		}
	}
}

// NextReader returns a reader for next frame.
// NextReader and SetReadDeadline needs be called sync.
//
// If Close called when NextReader,  it return io.EOF.
// Pause doesn't effect to NextReader. NextReader should wait till resumed
// and next FeedIn.
func (p *Payload) NextReader() (base.FrameType, base.PacketType, io.Reader, error) {
	ft, pt, r, err := p.decoder.NextReader()
	if err == io.EOF {
		arg, e := p.waitReader()
		if e != nil {
			return 0, 0, nil, e
		}
		p.decoder.FeedIn(arg.r, arg.supportBinary)
		ft, pt, r, err = p.decoder.NextReader()
	}
	if err != nil {
		return 0, 0, nil, p.store("next read", err)
	}
	return ft, pt, r, nil
}

// SetReadDeadline sets next reader deadline.
// NextReader and SetReadDeadline needs be called sync.
// NextReader will wait a FeedIn call, then it returns ReadCloser which
// decodes packet from FeedIn's Reader.
//
// If Close called when SetReadDeadline,  it return io.EOF.
// If beyond the time set by SetReadDeadline, it returns ErrTimeout.
// Pause doesn't effect to SetReadDeadline.
func (p *Payload) SetReadDeadline(t time.Time) error {
	p.readDeadline.Store(t)
	return nil
}

// NextWriter returns a writer for next frame.
// NextWriter and SetWriterDeadline needs be called sync.
// NextWriter will wait a FlushOut call, then it returns WriteCloser which
// encode package to FlushOut's Writer.
//
// If Close called when NextWriter,  it returns io.EOF.
// If beyond the time set by SetWriteDeadline, it returns ErrTimeout.
// If Pause called when NextWriter, it returns ErrPaused.
func (p *Payload) NextWriter(ft base.FrameType, pt base.PacketType) (io.WriteCloser, error) {
	return p.encoder.NextWriter(ft, pt)
}

// SetWriteDeadline sets next writer deadline.
// NextWriter and SetWriteDeadline needs be called sync.
//
// If Close called when SetWriteDeadline,  it return io.EOF.
// Pause doesn't effect to SetWriteDeadline.
func (p *Payload) SetWriteDeadline(t time.Time) error {
	p.writeDeadline.Store(t)
	return nil
}

// Pause pauses the payload. It will wait all reader and writer closed which
// created from NextReader or NextWriter.
// It can call in multi-goroutine.
func (p *Payload) Pause() {
	close(p.pauseChan())

	p.waiterLocker.Lock()
	p.pauseWaiter.Wait()
	p.waiterLocker.Unlock()
}

// Resume resumes the payload.
// It can call in multi-goroutine.
func (p *Payload) Resume() {
	p.pauseLocker.Lock()
	p.pause = make(chan struct{})
	p.pauseLocker.Unlock()
}

// Close closes the payload.
// It can call in multi-goroutine.
func (p *Payload) Close() error {
	p.closeOnce.Do(func() {
		close(p.close)
	})
	return nil
}

func (p *Payload) pauseChan() chan struct{} {
	p.pauseLocker.RLock()
	defer p.pauseLocker.RUnlock()
	return p.pause
}

func (p *Payload) readTimeout() (<-chan time.Time, bool) {
	deadline := p.readDeadline.Load().(time.Time)
	wait := deadline.Sub(time.Now())
	if deadline.IsZero() {
		// wait for every
		wait = math.MaxInt64
	}
	if wait <= 0 {
		return nil, false
	}
	return time.After(wait), true
}

func (p *Payload) writeTimeout() (<-chan time.Time, bool) {
	deadline := p.writeDeadline.Load().(time.Time)
	wait := deadline.Sub(time.Now())
	if deadline.IsZero() {
		// wait for every
		wait = math.MaxInt64
	}
	if wait <= 0 {
		return nil, false
	}
	return time.After(wait), true
}

func (p *Payload) waitReader() (readArg, error) {
	select {
	case <-p.close:
		return readArg{}, p.load()
	default:
	}

	if atomic.LoadInt64(&p.reading) == 1 {
		for {
			after, ok := p.readTimeout()
			if !ok {
				return readArg{}, p.store("read", errTimeout)
			}
			err := p.decoder.Close()
			select {
			case <-after:
				continue
			case p.readError <- err:
				fmt.Println("return reader")
				atomic.StoreInt64(&p.reading, 0)
			}
			break
		}
	}

	select {
	case <-p.close:
		return readArg{}, p.load()
	case <-p.pauseChan():
		return readArg{}, newOpError("payload", errPaused)
	default:
	}

	for {
		after, ok := p.readTimeout()
		if !ok {
			return readArg{}, p.store("read", errTimeout)
		}
		select {
		case <-p.close:
			return readArg{}, p.load()
		case <-p.pauseChan():
			return readArg{}, newOpError("payload", errPaused)
		case <-after:
			continue
		case arg := <-p.readerChan:
			fmt.Println("get reader")
			atomic.StoreInt64(&p.reading, 1)
			return arg, nil
		}
	}
}

func (p *Payload) beginWrite() (io.Writer, error) {
	select {
	case <-p.close:
		return nil, p.load()
	default:
	}
	select {
	case <-p.close:
		return nil, p.load()
	case <-p.pauseChan():
		return nil, newOpError("payload", errPaused)
	default:
	}

	for {
		after, ok := p.writeTimeout()
		if !ok {
			return nil, p.store("write", errTimeout)
		}
		select {
		case <-p.close:
			return nil, p.load()
		case <-p.pauseChan():
			return nil, newOpError("payload", errPaused)
		case <-after:
			continue
		case w := <-p.writerChan:
			return w, nil
		}
	}
}

func (p *Payload) endWrite(err error) error {
	select {
	case <-p.close:
		return p.load()
	default:
	}
	for {
		after, ok := p.writeTimeout()
		if !ok {
			return p.store("write", errTimeout)
		}
		ret := p.store("write", err)
		select {
		case <-p.close:
			return p.load()
		case <-after:
			continue
		case p.writeError <- err:
			return ret
		}
	}
}

func (p *Payload) store(op string, err error) error {
	old := p.err.Load()
	if old == nil {
		if err == io.EOF || err == nil {
			return err
		}
		op := newOpError(op, err)
		p.err.Store(op)
		return op
	}
	return old.(error)
}

func (p *Payload) load() error {
	ret := p.err.Load()
	if ret == nil {
		return io.EOF
	}
	return ret.(error)
}
