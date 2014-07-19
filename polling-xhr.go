package engineio

import (
	"io"
	"net/http"
	"time"
)

func init() {
	t := &polling{}
	RegisterTransport(t.Name(), t.HandlesUpgrades(), t.SupportsFraming(), NewPollingTransport)
}

type polling struct {
	sendChan     chan struct{}
	encoder      *PayloadEncoder
	socket       Conn
	pingInterval time.Duration
	pingTimeout  time.Duration
	lastPing     time.Time
	forceString  bool
	isClosed     bool
}

func NewPollingTransport(req *http.Request, pingInterval, pingTimeout time.Duration) (Transport, error) {
	ret := &polling{
		sendChan:     make(chan struct{}, 1),
		pingInterval: pingInterval,
		pingTimeout:  pingTimeout,
		lastPing:     time.Now(),
		forceString:  req.URL.Query()["b64"] != nil,
		isClosed:     false,
	}
	return ret, nil
}

func (*polling) Name() string {
	return "polling"
}

func (p *polling) SetSocket(s Conn) {
	p.socket = s
}

func (*polling) HandlesUpgrades() bool {
	return false
}

func (*polling) SupportsFraming() bool {
	return false
}

func (p *polling) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		p.get(w, r)
	case "POST":
		p.post(w, r)
	}
}

func (p *polling) Close() error {
	p.isClosed = true
	w, err := p.NextWriter(MessageText, CLOSE)
	if err != nil {
		return err
	}
	w.Close()
	return nil
}

func (p *polling) Upgraded() error {
	p.isClosed = true
	w, err := p.NextWriter(MessageText, UPGRADE)
	if err != nil {
		return err
	}
	w.Close()
	return nil
}

func (p *polling) NextWriter(msgType MessageType, packetType PacketType) (io.WriteCloser, error) {
	if p.isClosed {
		return nil, io.EOF
	}
	if p.encoder == nil {
		if p.forceString {
			p.encoder = NewStringPayloadEncoder()
		} else {
			p.encoder = NewBinaryPayloadEncoder()
		}
	}
	var ret io.WriteCloser
	var err error
	switch msgType {
	case MessageText:
		ret, err = p.encoder.NextString(packetType)
	case MessageBinary:
		ret, err = p.encoder.NextBinary(packetType)
	}
	if err != nil {
		return nil, err
	}
	return newPollingWriter(ret, p), nil
}

type pollingWriter struct {
	io.WriteCloser
	sendChan chan struct{}
}

func newPollingWriter(w io.WriteCloser, p *polling) *pollingWriter {
	return &pollingWriter{
		WriteCloser: w,
		sendChan:    p.sendChan,
	}
}

func (w *pollingWriter) Close() error {
	select {
	case w.sendChan <- struct{}{}:
	default:
	}
	return w.WriteCloser.Close()
}

func (p *polling) get(w http.ResponseWriter, r *http.Request) {
	if p.isClosed {
		http.Error(w, "socket closed", http.StatusBadGateway)
		return
	}
	defer func() {
		if p.isClosed {
			p.socket.onClose()
		}
	}()

	diff := time.Now().Sub(p.lastPing)
	pingTimeout := p.pingInterval - diff
	closeTimeout := p.pingTimeout - diff
	for {
		select {
		case <-p.sendChan:
			encoder := p.encoder
			p.encoder = nil
			w.Header().Set("Content-Type", "application/octet-stream")
			encoder.EncodeTo(w)
			return
		case <-time.After(pingTimeout):
			writer, err := p.NextWriter(MessageText, PING)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.Close()
		case <-time.After(closeTimeout):
			if err := p.Close(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

func (p *polling) post(w http.ResponseWriter, r *http.Request) {
	if p.isClosed {
		http.Error(w, "socket closed", http.StatusBadGateway)
		return
	}
	decoder := NewPayloadDecoder(r.Body)
	for {
		d, err := decoder.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch d.Type() {
		case PING:
			writer, err := p.NextWriter(MessageText, PONG)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.Close()
			fallthrough
		case PONG:
			p.lastPing = time.Now()
		case MESSAGE:
			p.socket.onMessage(d)
		case CLOSE:
			d.Close()
			p.Close()
			p.socket.onClose()
			break
		}
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("ok"))
}
