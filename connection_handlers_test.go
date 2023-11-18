package socketio

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/googollee/go-socket.io/engineio/session"
	"github.com/googollee/go-socket.io/parser"
)

type testStr struct {
	Result string `json:"result,omitempty"`
}

type fakeReader struct {
	data  [][]byte
	index int
	buf   *bytes.Buffer
}

func (r *fakeReader) NextReader() (session.FrameType, io.ReadCloser, error) {
	if r.index >= len(r.data) {
		return 0, nil, io.EOF
	}
	r.buf = bytes.NewBuffer(r.data[r.index])
	ft := session.BINARY
	if r.index == 0 {
		ft = session.TEXT
	}
	return ft, r, nil
}

func (r *fakeReader) Read(p []byte) (int, error) {
	return r.buf.Read(p)
}

func (r *fakeReader) Close() error {
	r.index++
	return nil
}

func TestAck(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	namespace := "/test"
	var id uint64 = 12
	c := &conn{
		handlers:   newNamespaceHandlers(),
		namespaces: newNamespaces(),
		decoder:    parser.NewDecoder(&fakeReader{data: [][]byte{[]byte("3-/test,12[{\"result\":\"pass\"}]")}}),
	}

	conn := newNamespaceConn(c, namespace, nil)
	c.namespaces.Set(namespace, conn)

	header := parser.Header{}

	called := false
	f := newAckFunc(func(t *testStr) {
		called = true
		should.Equal("pass", t.Result)
	})
	conn.ack.Store(id, f)

	event := "a"

	err := c.decoder.DecodeHeader(&header, &event)
	must.NoError(err)
	should.Equal(parser.Ack, header.Type)
	should.Equal(12, int(header.ID))
	should.Equal("/test", header.Namespace)
	err = ackPacketHandler(c, header)
	must.NoError(err)
	must.True(called)
}
