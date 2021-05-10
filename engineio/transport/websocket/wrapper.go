package websocket

import (
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/transport"
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

func (w wrapper) NextReader() (frame.Type, io.ReadCloser, error) {
	w.readLocker.Lock()
	defer w.readLocker.Unlock()

	// The wrapper remains locked until the returned ReadCloser is Closed.
	typ, r, err := w.Conn.NextReader()
	if err != nil {
		return 0, nil, err
	}

	switch typ {
	case websocket.TextMessage:
		return frame.String, newRcWrapper(w.readLocker, r), nil
	case websocket.BinaryMessage:
		return frame.Binary, newRcWrapper(w.readLocker, r), nil
	}

	return 0, nil, transport.ErrInvalidFrame
}

type rcWrapper struct {
	io.Reader
	nagTimer *time.Timer
	quitNag  chan struct{}
	l        *sync.Mutex
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
	r.l.Lock()
	defer r.l.Unlock()

	r.nagTimer.Stop()
	close(r.quitNag)
	// Attempt to drain the Reader.
	io.Copy(ioutil.Discard, r) // reader may be closed, ignore error
	// Unlock the wrapper's read lock for future calls to NextReader.
	return nil
}

func (w wrapper) NextWriter(FType frame.Type) (io.WriteCloser, error) {
	var t int

	switch FType {
	case frame.String:
		t = websocket.TextMessage
	case frame.Binary:
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
	io.WriteCloser
	nagTimer *time.Timer

	l       *sync.Mutex
	quitNag chan struct{}
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
