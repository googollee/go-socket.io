package engineio

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

type MessageType int

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

// Conn is the connection object of engine.io.
type Conn interface {

	// Id returns the session id of connection.
	Id() string

	// Request returns the first http request when established connection.
	Request() *http.Request

	// Close closes the connection.
	Close() error

	// NextReader returns the next message type, reader. If no message received, it will block.
	NextReader() (MessageType, io.ReadCloser, error)

	// NextWriter returns the next message writer with given message type.
	NextWriter(messageType MessageType) (io.WriteCloser, error)

	onOpen() error
	onPacket(r *packetDecoder)
	onClose()
	serveHTTP(w http.ResponseWriter, r *http.Request)
}

type connectionInfo struct {
	Sid          string        `json:"sid"`
	Upgrades     []string      `json:"upgrades"`
	PingInterval time.Duration `json:"pingInterval"`
	PingTimeout  time.Duration `json:"pingTimeout"`
}

type conn struct {
	id           string
	server       *Server
	t            transport
	readerChan   chan *connReader
	pingInterval time.Duration
	pingTimeout  time.Duration
	pingChan     chan bool
	req          *http.Request
	writerLocker sync.Mutex
	origin       transport
}

func newConn(id string, server *Server, transport transport, req *http.Request) (*conn, error) {
	ret := &conn{
		id:           id,
		server:       server,
		t:            transport,
		readerChan:   make(chan *connReader),
		pingInterval: server.config.PingInterval,
		pingTimeout:  server.config.PingTimeout,
		pingChan:     make(chan bool),
		req:          req,
	}
	transport.SetConn(ret)

	go ret.pingLoop()

	return ret, nil
}

func (s *conn) Id() string {
	return s.id
}

func (s *conn) Request() *http.Request {
	return s.req
}

func (s *conn) Close() error {
	if s.t == nil {
		return nil
	}
	s.writerLocker.Lock()
	w, err := s.t.NextWriter(MessageText, _CLOSE)
	if err != nil {
		return err
	}
	w.Close()
	s.writerLocker.Unlock()
	if s.origin != nil {
		s.origin.Close()
		s.origin = nil
	}
	err = s.t.Close()
	s.t = nil
	return err
}

func (s *conn) NextReader() (MessageType, io.ReadCloser, error) {
	reader := <-s.readerChan
	if reader == nil {
		return MessageText, nil, io.EOF
	}

	return reader.MessageType(), reader, nil
}

func (s *conn) NextWriter(messageType MessageType) (io.WriteCloser, error) {
	s.writerLocker.Lock()
	ret, err := s.t.NextWriter(messageType, _MESSAGE)
	if err != nil {
		s.writerLocker.Unlock()
		return nil, err
	}
	return newConnWriter(ret, &s.writerLocker), nil
}

func (s *conn) nextWriter(messageType MessageType, packetType packetType) (io.WriteCloser, error) {
	s.writerLocker.Lock()
	ret, err := s.t.NextWriter(messageType, packetType)
	if err != nil {
		s.writerLocker.Unlock()
		return nil, err
	}
	return newConnWriter(ret, &s.writerLocker), nil
}

func (s *conn) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if s.t == nil {
		http.Error(w, "closed", http.StatusBadRequest)
		return
	}
	transportName := r.URL.Query().Get("transport")
	if s.t.Name() != transportName {
		if !s.server.config.AllowUpgrades {
			http.Error(w, "not allow upgrade", http.StatusBadRequest)
			return
		}
		if s.origin != nil && s.origin.Name() == transportName {
			s.origin.ServeHTTP(w, r)
			return
		}

		creater := s.server.transports.GetUpgrade(transportName)
		if creater == nil {
			http.Error(w, "invalid transport", http.StatusBadRequest)
			return
		}
		transport, err := creater(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		transport.SetConn(s)
		if s.origin != nil {
			s.t.Close()
		} else {
			s.origin = s.t
		}
		s.t = transport
	}

	s.t.ServeHTTP(w, r)
}

func (s *conn) onOpen() error {
	resp := connectionInfo{
		Sid:          s.id,
		Upgrades:     s.server.transports.Upgrades(),
		PingInterval: s.server.config.PingInterval / time.Millisecond,
		PingTimeout:  s.server.config.PingTimeout / time.Millisecond,
	}
	w, err := s.t.NextWriter(MessageText, _OPEN)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(resp); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func (s *conn) onPacket(decoder *packetDecoder) {
	switch decoder.Type() {
	case _PING:
		if s.origin != nil {
			if w, _ := s.origin.NextWriter(MessageText, _NOOP); w != nil {
				w.Close()
			}
		}
		if w, _ := s.nextWriter(MessageText, _PONG); w != nil {
			io.Copy(w, decoder)
			w.Close()
		}
		fallthrough
	case _PONG:
		s.pingChan <- true
		return
	case _CLOSE:
		s.Close()
		s.onClose()
		return
	case _UPGRADE:
		s.origin.Close()
		s.origin = nil
		return
	case _NOOP:
		return
	}

	closeChan := make(chan struct{})
	s.readerChan <- newConnReader(decoder, closeChan)
	<-closeChan
	close(closeChan)
}

func (s *conn) onClose() {
	close(s.readerChan)
	close(s.pingChan)
	s.server.onClose(s)
	s.origin = nil
	s.t = nil
}

func (s *conn) pingLoop() {
	lastPing := time.Now()
	for {
		now := time.Now()
		diff := now.Sub(lastPing)
		select {
		case ping := <-s.pingChan:
			if !ping {
				return
			}
			lastPing = time.Now()
		// case <-time.After(s.pingInterval - diff):
		// 	s.writerLocker.Lock()
		// 	if w, _ := s.t.NextWriter(MessageText, _PING); w != nil {
		// 		w.Close()
		// 	}
		// 	s.writerLocker.Unlock()
		case <-time.After(s.pingTimeout - diff):
			s.Close()
			s.onClose()
			return
		}
	}
}
