package engineio

import (
	"io"
	"net/http"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
)

func init() {
	t := &websocket{}
	RegisterTransport(t.Name(), t.HandlesUpgrades(), t.SupportsFraming(), NewWebsocketTransport)
}

type websocket struct {
	socket       Conn
	conn         *ws.Conn
	quitChan     chan struct{}
	pingChan     chan time.Time
	pingInterval time.Duration
	pingTimeout  time.Duration
	lastPing     time.Time
	isClosed     bool
	writeLocker  sync.Mutex
}

func NewWebsocketTransport(req *http.Request, pingInterval, pingTimeout time.Duration) (Transport, error) {
	ret := &websocket{
		quitChan:     make(chan struct{}, 1),
		pingChan:     make(chan time.Time),
		pingInterval: pingInterval,
		pingTimeout:  pingTimeout,
		lastPing:     time.Now(),
		isClosed:     false,
	}
	return ret, nil
}

func (*websocket) Name() string {
	return "websocket"
}

func (p *websocket) SetSocket(s Conn) {
	p.socket = s
}

func (*websocket) HandlesUpgrades() bool {
	return true
}

func (*websocket) SupportsFraming() bool {
	return true
}

func (p *websocket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("transport")
	if name != p.Name() {
		encoder := NewBinaryPayloadEncoder()
		writer, err := encoder.NextString(NOOP)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Close()
		encoder.EncodeTo(w)
		return
	}

	conn, err := ws.Upgrade(w, r, nil, 10240, 10240)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	p.conn = conn
	defer func() {
		close(p.quitChan)
		p.conn.Close()
		p.socket.onClose()
	}()

	go func(lastPing time.Time) {
		for {
			diff := time.Now().Sub(lastPing)
			select {
			case lastPing = <-p.pingChan:
			case <-time.After(p.pingInterval - diff):
				w, _ := p.NextWriter(MessageText, PING)
				w.Close()
			case <-time.After(p.pingTimeout - diff):
				return
			case <-p.quitChan:
				return
			}
		}
	}(p.lastPing)

	for {
		t, r, err := conn.NextReader()
		if err != nil {
			return
		}

		if t == ws.TextMessage || t == ws.BinaryMessage {
			decoder, _ := NewDecoder(r)
			switch decoder.Type() {
			case PING:
				w, _ := p.NextWriter(MessageText, PONG)
				io.Copy(w, decoder)
				w.Close()
				fallthrough
			case PONG:
				p.lastPing = time.Now()
				p.pingChan <- p.lastPing
			case CLOSE:
				p.Close()
				return
			case UPGRADE:
			case NOOP:
			default:
				p.socket.onMessage(decoder)
			}
			decoder.Close()
		}
	}
}

type websocketWriter struct {
	io.WriteCloser
	locker *sync.Mutex
}

func newWebsocketWriter(w io.WriteCloser, locker *sync.Mutex) websocketWriter {
	return websocketWriter{
		WriteCloser: w,
		locker:      locker,
	}
}

func (p *websocket) NextWriter(msgType MessageType, packetType PacketType) (io.WriteCloser, error) {
	if p.isClosed {
		return nil, io.EOF
	}
	if packetType == CLOSE {
		ret, err := p.conn.NextWriter(ws.CloseMessage)
		if err != nil {
			return nil, err
		}
		return newWebsocketWriter(ret, &p.writeLocker), nil
	}
	var ret io.WriteCloser
	var err error
	switch msgType {
	case MessageText:
		ret, err = p.conn.NextWriter(ws.TextMessage)
		if err != nil {
			return nil, err
		}
		ret, err = NewStringEncoder(ret, packetType)
	case MessageBinary:
		ret, err = p.conn.NextWriter(ws.BinaryMessage)
		if err != nil {
			return nil, err
		}
		ret, err = NewBinaryEncoder(ret, packetType)
	}
	if err != nil {
		return nil, err
	}
	return newWebsocketWriter(ret, &p.writeLocker), nil
}

func (p *websocket) Close() error {
	w, err := p.NextWriter(MessageText, CLOSE)
	if err != nil {
		return err
	}
	w.Close()
	p.isClosed = true
	p.conn.Close()
	return nil
}

func (p websocket) Upgraded() error {
	return nil
}
