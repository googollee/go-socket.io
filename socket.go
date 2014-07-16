package engineio

import (
	"io"
	"net/http"
)

type Socket interface {
	Request() *http.Request
	Upgraded() bool
	Close() error
	NextReader() (io.ReadCloser, error)
	NextWriter(messageType MessageType) (io.WriteCloser, error)
	// SetReadDeadline(t time.Time) error
	// SetWriteDeadline(t time.Time) error
	// ReadJSON(v interface{}) error
	// WriteJSON(v interface{}) error

	transport() Transport
	upgrade(transport Transport)
	onMessage(r io.Reader)
	onClose()
}

type socket struct {
	id         string
	server     *Server
	t          Transport
	upgraded   bool
	isClosed   bool
	readerChan chan io.ReadCloser
	req        *http.Request
}

func newSocket(id string, server *Server, transport Transport, req *http.Request) *socket {
	ret := &socket{
		id:         id,
		server:     server,
		t:          transport,
		upgraded:   false,
		isClosed:   false,
		readerChan: make(chan io.ReadCloser),
		req:        req,
	}
	transport.SetSocket(ret)

	return ret
}

func (s *socket) Request() *http.Request {
	return s.req
}

func (s *socket) Upgraded() bool {
	return s.upgraded
}

func (s *socket) Close() error {
	if s.isClosed {
		return nil
	}
	s.isClosed = true
	return s.transport().Close()
}

type connReaer struct {
	io.Reader
	closeChan chan struct{}
}

func (r *connReaer) Close() error {
	if r.closeChan == nil {
		return nil
	}
	r.closeChan <- struct{}{}
	r.closeChan = nil
	return nil
}

func (s *socket) NextReader() (io.ReadCloser, error) {
	if s.isClosed {
		return nil, io.EOF
	}
	reader := <-s.readerChan
	if reader == nil {
		return nil, io.EOF
	}

	return reader, nil
}

func (s *socket) NextWriter(messageType MessageType) (io.WriteCloser, error) {
	if s.isClosed {
		return nil, io.EOF
	}
	return s.transport().NextWriter(messageType, MESSAGE)
}

func (s *socket) transport() Transport {
	return s.t
}

func (s *socket) upgrade(transport Transport) {
	s.t.Upgraded()
	transport.SetSocket(s)
	s.t = transport
	s.upgraded = true
}

func (s *socket) onMessage(r io.Reader) {
	if s.isClosed {
		return
	}

	closeChan := make(chan struct{})
	reader := &connReaer{
		Reader:    r,
		closeChan: closeChan,
	}
	s.readerChan <- reader
	<-closeChan
	close(closeChan)
}

func (s *socket) onClose() {
	s.isClosed = true
	close(s.readerChan)
	s.server.onClose(s)
}
