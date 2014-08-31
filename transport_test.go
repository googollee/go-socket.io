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
	t1 := newFakeTransportCreater(true, "t1")
	t2 := newFakeTransportCreater(true, "t2")
	t3 := newFakeTransportCreater(false, "t3")
	registerTransport("t1", true, t1)
	registerTransport("t2", false, t2)
	registerTransport("t3", true, t3)
	tt, _ := newTransportsType([]string{"t1", "t2"})

	Convey("Create transports type", t, func() {
		t, err := newTransportsType(nil)
		So(err, ShouldBeNil)
		So(len(t), ShouldEqual, 5)
		var names []string
		for n := range t {
			names = append(names, n)
		}
		So(names, ShouldContain, "t1")
		So(names, ShouldContain, "t2")
		So(names, ShouldContain, "t3")
		So(names, ShouldContain, "polling")
		So(names, ShouldContain, "websocket")
		_, err = newTransportsType([]string{"t1", "t2"})
		So(err, ShouldBeNil)
		_, err = newTransportsType([]string{"t1", "nonexist"})
		So(err.Error(), ShouldEqual, "invalid transport name nonexist")
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
