package websocket

import (
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/googollee/go-engine.io/transport"
	"github.com/gorilla/websocket"
)

type wrapper struct {
	*websocket.Conn
	writeLocker *sync.Mutex
	readLocker  *sync.Mutex
}

func newWrapper(conn *websocket.Conn) wrapper {
	return wrapper{
		Conn:        conn,
		writeLocker: new(sync.Mutex),
		readLocker:  new(sync.Mutex),
	}
}

func (w wrapper) NextReader() (base.FrameType, io.ReadCloser, error) {
	w.readLocker.Lock()
	typ, r, err := w.Conn.NextReader()
	// The wrapper remains locked until the returned ReadCloser is Closed.
	if err != nil {
		w.readLocker.Unlock()
		return 0, nil, err
	}
	switch typ {
	case websocket.TextMessage:
		return base.FrameString, newRcWrapper(w.readLocker, r), nil
	case websocket.BinaryMessage:
		return base.FrameBinary, newRcWrapper(w.readLocker, r), nil
	}
	w.readLocker.Unlock()
	return 0, nil, transport.ErrInvalidFrame
}

type rcWrapper struct {
	nagTimer *time.Timer
	quitNag  chan struct{}
	l        *sync.Mutex
	io.Reader
}

func newRcWrapper(l *sync.Mutex, r io.Reader) rcWrapper {
	timer := time.NewTimer(30 * time.Second)
	q := make(chan struct{})
	go func() {
		select {
		case <-q:
		case <-timer.C:
			fmt.Println("Did you forget to Close() the ReadCloser from NextReader?")
		}
	}()
	return rcWrapper{
		nagTimer: timer,
		quitNag:  q,
		l:        l,
		Reader:   r,
	}
}

func (r rcWrapper) Close() error {
	// Stop the nagger.
	r.nagTimer.Stop()
	close(r.quitNag)
	// Attempt to drain the Reader.
	io.Copy(ioutil.Discard, r) // reader may be closed, ignore error
	// Unlock the wrapper's read lock for future calls to NextReader.
	r.l.Unlock()
	return nil
}

func (w wrapper) NextWriter(typ base.FrameType) (io.WriteCloser, error) {
	var t int
	switch typ {
	case base.FrameString:
		t = websocket.TextMessage
	case base.FrameBinary:
		t = websocket.BinaryMessage
	default:
		return nil, transport.ErrInvalidFrame
	}

	w.writeLocker.Lock()
	writer, err := w.Conn.NextWriter(t)
	// The wrapper remains locked until the returned WriteCloser is Closed.
	if err != nil {
		w.writeLocker.Unlock()
		return nil, err
	}

	return newWcWrapper(w.writeLocker, writer), nil
}

type wcWrapper struct {
	nagTimer *time.Timer
	quitNag  chan struct{}
	l        *sync.Mutex
	io.WriteCloser
}

func newWcWrapper(l *sync.Mutex, w io.WriteCloser) wcWrapper {
	timer := time.NewTimer(30 * time.Second)
	q := make(chan struct{})
	go func() {
		select {
		case <-q:
		case <-timer.C:
			fmt.Println("Did you forget to Close() the WriteCloser from NextWriter?")
		}
	}()
	return wcWrapper{
		nagTimer:    timer,
		quitNag:     q,
		l:           l,
		WriteCloser: w,
	}
}

func (w wcWrapper) Close() error {
	// Stop the nagger.
	w.nagTimer.Stop()
	close(w.quitNag)
	// Unlock the wrapper's write lock for future calls to NextWriter.
	defer w.l.Unlock()
	return w.WriteCloser.Close()
}
