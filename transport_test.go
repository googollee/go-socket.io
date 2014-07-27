package engineio

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type fakeTransport struct {
	name     string
	conn     Conn
	isClosed bool
	encoder  *payloadEncoder
}

func newFakeTransportCreater(ok bool, name string) transportCreateFunc {
	return func(*http.Request) (transport, error) {
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

func (t *fakeTransport) SetConn(conn Conn) {
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

func TestTransport(t *testing.T) {
	tt := transportsType{}
	t1 := newFakeTransportCreater(true, "t1")
	t2 := newFakeTransportCreater(true, "t2")

	Convey("Test register", t, func() {
		tt.Register("t1", true, t1)
		tt.Register("t2", false, t2)
	})

	Convey("Test upgrades", t, func() {
		So(tt.Upgrades(), ShouldResemble, []string{"t1"})
	})

	Convey("Test get creater", t, func() {
		So(tt.GetCreater("t1"), ShouldEqual, t1)
		So(tt.GetCreater("t2"), ShouldEqual, t2)
		So(tt.GetCreater("nonexit"), ShouldBeNil)
	})

	Convey("Test get upgrade", t, func() {
		So(tt.GetUpgrade("t1"), ShouldEqual, t1)
		So(tt.GetUpgrade("t2"), ShouldBeNil)
		So(tt.GetUpgrade("nonexit"), ShouldBeNil)
	})

}
