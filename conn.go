package engineio

import (
	"io"
	"net/http"
)

type MessageType int // func(*PayloadEncoder, PacketType) (io.WriteCloser, error)

const (
	MessageText MessageType = iota
	MessageBinary
)

func (t MessageType) String() string {
	switch t {
	case MessageText:
		return "message text"
	case MessageBinary:
		return "message binary"
	}
	return "message known"
}

type Conn interface {
	Id() string
	Request() *http.Request
	Upgraded() bool
	Close() error
	NextReader() (MessageType, io.ReadCloser, error)
	NextWriter(messageType MessageType) (io.WriteCloser, error)
	// SetReadDeadline(t time.Time) error
	// SetWriteDeadline(t time.Time) error
	// ReadJSON(v interface{}) error
	// WriteJSON(v interface{}) error

	transport() Transport
	upgrade(transport Transport)
	onMessage(r *PacketDecoder)
	onClose()
}

type conn struct {
	id         string
	server     *Server
	t          Transport
	upgraded   bool
	isClosed   bool
	readerChan chan *connReader
	req        *http.Request
}

func newSocket(id string, server *Server, transport Transport, req *http.Request) *conn {
	ret := &conn{
		id:         id,
		server:     server,
		t:          transport,
		upgraded:   false,
		isClosed:   false,
		readerChan: make(chan *connReader),
		req:        req,
	}
	transport.SetSocket(ret)

	return ret
}

func (s *conn) Id() string {
	return s.id
}

func (s *conn) Request() *http.Request {
	return s.req
}

func (s *conn) Upgraded() bool {
	return s.upgraded
}

func (s *conn) Close() error {
	if s.isClosed {
		return nil
	}
	s.isClosed = true
	return s.transport().Close()
}

type connReader struct {
	*PacketDecoder
	closeChan chan struct{}
}

func (r *connReader) Close() error {
	if r.closeChan == nil {
		return nil
	}
	r.closeChan <- struct{}{}
	r.closeChan = nil
	return nil
}

func (s *conn) NextReader() (MessageType, io.ReadCloser, error) {
	if s.isClosed {
		return MessageText, nil, io.EOF
	}
	reader := <-s.readerChan
	if reader == nil {
		return MessageText, nil, io.EOF
	}

	return reader.MessageType(), reader, nil
}

func (s *conn) NextWriter(messageType MessageType) (io.WriteCloser, error) {
	if s.isClosed {
		return nil, io.EOF
	}
	return s.transport().NextWriter(messageType, MESSAGE)
}

func (s *conn) transport() Transport {
	return s.t
}

func (s *conn) upgrade(transport Transport) {
	s.t.Upgraded()
	transport.SetSocket(s)
	s.t = transport
	s.upgraded = true
}

func (s *conn) onMessage(decoder *PacketDecoder) {
	if s.isClosed {
		return
	}

	closeChan := make(chan struct{})
	reader := &connReader{
		PacketDecoder: decoder,
		closeChan:     closeChan,
	}
	s.readerChan <- reader
	<-closeChan
	close(closeChan)
}

func (s *conn) onClose() {
	s.isClosed = true
	close(s.readerChan)
	s.server.onClose(s)
}
