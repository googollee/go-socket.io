package engineio

import (
	"bytes"
	"io"
)

type fakeTransport struct {
	name     string
	callback connCallback
	isClosed bool
	encoder  *payloadEncoder
}

func newFakeTransportCreater(ok bool, name string) transportCreateFunc {
	return func(http.ResponseWriter, *http.Request) (transport, error) {
		if !ok {
			return nil, fmt.Errorf("transport %s error", name)
		}
		return &fakeTransport{
			name:    name,
			encoder: newStringPayloadEncoder(),
		}, nil
	}
}

func (t *fakeTransport) Name() string {
	return t.name
}

func (t *fakeTransport) SetConn(conn connCallback) {
	t.conn = conn
}

func (t *fakeTransport) ServeHTTP(http.ResponseWriter, *http.Request) {
}

func (t *fakeTransport) Close() error {
	t.isClosed = true
	return nil
}

func (t *fakeTransport) NextWriter(messageType MessageType, packetType packetType) (io.WriteCloser, error) {
	if messageType == MessageText {
		return t.encoder.NextString(packetType)
	}
	return t.encoder.NextBinary(packetType)
}
