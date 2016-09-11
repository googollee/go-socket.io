// Package payload is framing layer for connection which doesn't support framing.
package payload

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/googollee/go-engine.io/base"
)

// ErrTimeout is timeout error.
var ErrTimeout = timeoutError{}

// ErrPaused means decoder is paused.
var ErrPause = errors.New("pause")

// ByteReader can read byte by byte
type ByteReader interface {
	ReadByte() (byte, error)
	io.Reader
}

// ByteWriter can write byte by byte
type ByteWriter interface {
	WriteByte(b byte) error
	io.Writer
}

// Signal sends signal to encoder/decoder.
type Signal struct {
	close chan struct{}

	pauseLocker sync.RWMutex
	pause       chan struct{}

	errLocker sync.RWMutex
	err       error
}

// NewSignal creates a new signal.
func NewSignal() *Signal {
	return &Signal{
		close: make(chan struct{}),
		pause: make(chan struct{}),
	}
}

// Close sends close signal.
func (s *Signal) Close() {
	close(s.close)
}

// WaitClose waits close signal with chan.
func (s *Signal) WaitClose() <-chan struct{} {
	return s.close
}

// Pause sends pause signal.
func (s *Signal) Pause() {
	s.pauseLocker.Lock()
	defer s.pauseLocker.Unlock()
	close(s.pause)
}

// WaitPause waits pause signal with chan.
func (s *Signal) WaitPause() <-chan struct{} {
	s.pauseLocker.RLock()
	defer s.pauseLocker.RUnlock()
	return s.pause
}

// Resume resumes from pause status.
func (s *Signal) Resume() {
	s.pauseLocker.Lock()
	defer s.pauseLocker.Unlock()
	s.pause = make(chan struct{})
}

// StoreError saves error.
func (s *Signal) StoreError(err error) error {
	s.errLocker.Lock()
	defer s.errLocker.Unlock()
	s.err = err
	return err
}

// LoadError loads error.
func (s *Signal) LoadError() error {
	s.errLocker.RLock()
	defer s.errLocker.RUnlock()
	if s.err == nil {
		return io.EOF
	}
	return s.err
}

// Encoder encodes packet frames into a payload. It need be closed before
// sending payload data.
// It can changing output Writer w while using. The senario is, when using xhr
// as connection, it need change BufWriter as output between GET response. It
// must close frame and Flushed before SetWriter.
type Encoder interface {
	base.FrameWriter
	FlushOut(w io.Writer) error
	SetDeadline(t time.Time) error
}

// NewEncoder creates a new encoder, output to w. The maximum size of one frame
// is limited with maxFrameSize. If w supports binary, set supportBinary true,
// otherwise set it to false.
func NewEncoder(supportBinary bool, sig *Signal) Encoder {
	return newEncoder(supportBinary, sig)
}

// ErrInvalidPayload is error of invalid payload while decoding.
var ErrInvalidPayload = errors.New("invalid payload")

// Decoder decodes packet from a payload.
// It can be changed input BufReader r while using. The senario is, when using
// xhr as connection, it need change request body as input between POST request.
// It must close frame before SetReader.
type Decoder interface {
	base.FrameReader
	FeedIn(typ base.FrameType, r io.Reader) error
	SetDeadline(t time.Time) error
}

// NewDecoder creates a new decoder, input from r. If r supports binary, set
// supportBinary true, otherwise set it to false.
func NewDecoder(sig *Signal) Decoder {
	return newDecoder(sig)
}
