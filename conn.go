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

type conn struct {
	id           string
	server       *Server
	t            Transport
	readerChan   chan *connReader
	pingInterval time.Duration
	pingTimeout  time.Duration
	pingChan     chan bool
	req          *http.Request
	writerLocker sync.Mutex
	origin       Transport
}

func newSocket(id string, server *Server, transport Transport, req *http.Request) (*conn, error) {
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
	err := ret.onOpen()
	if err != nil {
		return nil, err
	}

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
	s.writerLocker.Lock()
	w, err := s.t.NextWriter(MessageText, CLOSE)
	if err != nil {
		return err
	}
	w.Close()
	s.writerLocker.Unlock()
	if s.origin != nil {
		s.origin.Close()
	}
	return s.t.Close()
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
	ret, err := s.t.NextWriter(messageType, MESSAGE)
	if err != nil {
		s.writerLocker.Unlock()
		return nil, err
	}
	return newConnWriter(ret, &s.writerLocker), nil
}

func (s *conn) nextWriter(messageType MessageType, packetType PacketType) (io.WriteCloser, error) {
	s.writerLocker.Lock()
	ret, err := s.t.NextWriter(messageType, packetType)
	if err != nil {
		s.writerLocker.Unlock()
		return nil, err
	}
	return newConnWriter(ret, &s.writerLocker), nil
}

func (s *conn) serveHTTP(w http.ResponseWriter, r *http.Request) {
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

		creater := getTransportUpgrade(transportName)
		if creater == nil {
			http.Error(w, "invalid transport", http.StatusBadRequest)
			return
		}
		transport, err := creater(r)
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
	resp := struct {
		Sid          string        `json:"sid"`
		Upgrades     []string      `json:"upgrades"`
		PingInterval time.Duration `json:"pingInterval"`
		PingTimeout  time.Duration `json:"pingTimeout"`
	}{
		Sid:          s.id,
		Upgrades:     getUpgradesHandlers(),
		PingInterval: s.server.config.PingInterval / time.Millisecond,
		PingTimeout:  s.server.config.PingTimeout / time.Millisecond,
	}
	w, err := s.t.NextWriter(MessageText, OPEN)
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
	case PING:
		if s.origin != nil {
			if w, _ := s.origin.NextWriter(MessageText, NOOP); w != nil {
				w.Close()
			}
		}
		if w, _ := s.nextWriter(MessageText, PONG); w != nil {
			io.Copy(w, decoder)
			w.Close()
		}
		fallthrough
	case PONG:
		s.pingChan <- true
		return
	case CLOSE:
		s.Close()
		s.onClose()
		return
	case UPGRADE:
		s.origin.Close()
		s.origin = nil
		return
	case NOOP:
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
		case <-time.After(s.pingInterval - diff):
			s.writerLocker.Lock()
			if w, _ := s.t.NextWriter(MessageText, PING); w != nil {
				w.Close()
			}
			s.writerLocker.Unlock()
		case <-time.After(s.pingTimeout - diff):
			s.Close()
			s.onClose()
			return
		}
	}
}
