package engineio

import (
	"io"
	"net/http"
)

func init() {
	registerTransport("polling", false, newPollingTransport)
}

type polling struct {
	sendChan chan bool
	encoder  *payloadEncoder
	conn     Conn
}

func newPollingTransport(req *http.Request) (transport, error) {
	newEncoder := newBinaryPayloadEncoder
	if req.URL.Query()["b64"] != nil {
		newEncoder = newStringPayloadEncoder
	}
	ret := &polling{
		sendChan: make(chan bool, 1),
		encoder:  newEncoder(),
	}
	return ret, nil
}

func (*polling) Name() string {
	return "polling"
}

func (p *polling) SetConn(s Conn) {
	p.conn = s
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
	close(p.sendChan)
	p.sendChan = nil
	p.conn = nil
	return nil
}

func (p *polling) NextWriter(msgType MessageType, packetType packetType) (io.WriteCloser, error) {
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
	sendChan chan bool
}

func newPollingWriter(w io.WriteCloser, p *polling) *pollingWriter {
	return &pollingWriter{
		WriteCloser: w,
		sendChan:    p.sendChan,
	}
}

func (w *pollingWriter) Close() error {
	select {
	case w.sendChan <- true:
	default:
	}
	return w.WriteCloser.Close()
}

func (p *polling) get(w http.ResponseWriter, r *http.Request) {
	send := <-p.sendChan
	if !send {
		http.Error(w, "closed", http.StatusBadRequest)
		return
	}

	if p.encoder.IsString() {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	p.encoder.EncodeTo(w)
}

func (p *polling) post(w http.ResponseWriter, r *http.Request) {
	if p.conn == nil {
		http.Error(w, "closed", http.StatusBadRequest)
		return
	}
	decoder := newPayloadDecoder(r.Body)
	for {
		d, err := decoder.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p.conn.onPacket(d)
		d.Close()
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("ok"))
}
