package engineio

import (
	"fmt"
	"github.com/googollee/go-engine.io/message"
	"io"
	"net/http"
	"time"

	"github.com/googollee/go-engine.io/parser"
	"github.com/googollee/go-engine.io/transport"
)

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
}

type state int

const (
	stateUnknow state = iota
	stateNormal
	stateUpgrading
	stateClosing
	stateClosed
)

type serverConn struct {
	id           string
	current      transport.Server
	upgrading    transport.Server
	callback     serverCallback
	request      *http.Request
	state        state
	readerChan   chan io.ReadCloser
	pingTimeout  time.Duration
	pingInterval time.Duration
	pingChan     chan bool
}

func newServerConn(id string, w http.ResponseWriter, r *http.Request, callback serverCallback) (*serverConn, error) {
	transportName := r.URL.Query().Get("transport")
	creater := getCreater(transportName)
	if creater != nil {
		return nil, fmt.Errorf("invalid transport %s", transportName)
	}
	ret := &serverConn{
		id:           id,
		callback:     callback,
		request:      r,
		state:        stateNormal,
		readerChan:   make(chan io.ReadCloser),
		pingTimeout:  callback.Config().PingTimeout,
		pingInterval: callback.Config().PingInterval,
		pingChan:     make(chan bool),
	}
	transport, err := creater.Server(w, r, ret)
	if err != nil {
		return nli, err
	}
	ret.current = transport

	go ret.pingLoop()

	return ret, nil
}

func (c *serverConn) Id() string {
	return c.id
}

func (c *serverConn) Request() *http.Request {
	return c.request
}

func (c *serverConn) NextReader() (io.ReadCloser, error) {
	if c.getState() == stateClosed {
		return nil, io.EOF
	}
	ret := <-c.readerChan
	if ret == nil {
		return nil, io.EOF
	}
	return ret, nil
}

func (c *serverConn) NextWriter(t message.MessageType) (io.WriteCloser, error) {
	if c.getState() != stateNormal {
		return nil, io.EOF
	}
	ret, err := c.current.NextWriter(t, parser.MESSAGE)
	return ret, err
}

func (c *serverConn) Close() error {
	if c.getState() != stateNormal {
		return nil
	}
	if c.upgrading != nil {
		c.upgrading.Close()
	}
	if w, err := c.current.NextWriter(message.MessageText, parser.CLOSE); err == nil {
		w.Close()
	}
	if err := c.current.Close(); err != nil {
		return err
	}
	c.setState(stateClosing)
	return nil
}

func (c *serverConn) OnPacket(r *parser.PacketDecoder) {
	switch r.Type() {
	case parser.OPEN:
	case parser.CLOSE:
		c.current.Close()
	case parser.PING:
		newWriter := c.current.NextWriter
		if c.upgrading != nil {
			if w, _ := c.current(message.MessageText, parser.NOOP); w != nil {
				w.Close()
			}
			newWriter = c.upgrading.NextWriter
		}
		if w, _ := newWriter(message.MessageText, parser.PONG); w != nil {
			io.Copy(w, r)
			w.Close()
		}
		fallthrough
	case parser.PONG:
		c.pingChan <- true
	case parser.MESSAGE:
		closeChan := make(chan struct{})
		s.readerChan <- newConnReader(r, closeChan)
		<-closeChan
		close(closeChan)
		r.Close()
	case parser.UPGRADE:
		c.current.Close()
		c.current = c.upgrading
		c.upgrading = nil
	case parser.NOOP:
	}
}

func (c *serverConn) OnClose(server transport.Server) {
	if server == c.upgrading {
		c.upgrading = nil
		return
	}
	if server != c.current {
		return
	}
	if c.upgrading != nil {
		c.upgrading.Close()
		c.upgrading = nil
	}
	close(c.readerChan)
	close(c.pingChan)
	c.callback.OnClose(c.id)
}

func (c *serverConn) getState() state {
	return c.state
}

func (c *serverConn) setState(state state) {
	c.state = state
}

func (c *serverConn) pingLoop() {
	last := time.Now()
	for {
		now := time.Now()
		diff := now.Sub(last)
		select {
		case ok := <-c.pingChan:
			if !ok {
				return
			}
			last = time.Now()
		// case <-time.After(c.pingInterval - diff):
		// 	c.writerLocker.Lock()
		// 	if w, _ := c.t.NextWriter(MessageText, _PING); w != nil {
		// 		w.Close()
		// 	}
		// 	c.writerLocker.Unlock()
		case <-time.After(c.pingTimeout - diff):
			c.Close()
			return
		}
	}
}

type serverCallback interface {
	Config() config
	Transports() transportsType
	OnClose(sid string)
}
