package engineio

import (
	"io"
	"net/http"
	"time"

	ws "github.com/gorilla/websocket"
)

func init() {
	t := &websocket{}
	RegisterTransport(t.Name(), t.HandlesUpgrades(), t.SupportsFraming(), NewWebsocketTransport)
}

type websocket struct {
	socket       Socket
	conn         *ws.Conn
	quitChan     chan struct{}
	pingInterval time.Duration
	pingTimeout  time.Duration
	lastPing     time.Time
	isClosed     bool
}

func NewWebsocketTransport(req *http.Request, pingInterval, pingTimeout time.Duration) (Transport, error) {
	ret := &websocket{
		quitChan:     make(chan struct{}, 1),
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

func (p *websocket) SetSocket(s Socket) {
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
		p.conn.Close()
		p.socket.onClose()
	}()

	go func() {
		defer func() {
			if !p.isClosed {
				p.isClosed = true
				close(p.quitChan)
			}
		}()
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
	}()

	for {
		diff := time.Now().Sub(p.lastPing)
		select {
		case <-time.After(p.pingInterval - diff):
			w, _ := p.NextWriter(MessageText, PING)
			w.Close()
		case <-time.After(p.pingTimeout - diff):
			return
		case <-p.quitChan:
			return
		}
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
		return ret, nil
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
	return ret, nil
}

func (p *websocket) Close() error {
	w, err := p.NextWriter(MessageText, CLOSE)
	if err != nil {
		return err
	}
	w.Close()
	p.isClosed = true
	close(p.quitChan)
	return nil
}

func (p websocket) Upgraded() error {
	return nil
}
