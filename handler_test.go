package socketio

import (
	"testing"

	"github.com/googollee/go-engine.io"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"net/http"
)

type FakeBroadcastAdaptor struct{}

func (f *FakeBroadcastAdaptor) Join(room string, socket Socket) error {
	return nil
}

func (f *FakeBroadcastAdaptor) Leave(room string, socket Socket) error {
	return nil
}

func (f *FakeBroadcastAdaptor) Send(ignore Socket, room, event string, args ...interface{}) error {
	return nil
}

func (f *FakeBroadcastAdaptor) Len(room string) int {
	return 0
}

type FakeReadCloser struct{}

func (fr *FakeReadCloser) Read(p []byte) (n int, err error) {
	p = append(p, byte(128))
	return 1, nil
}

func (fr *FakeReadCloser) Close() error {
	return nil
}

type FakeWriteCloser struct{}

func (fr *FakeWriteCloser) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (fr *FakeWriteCloser) Close() error {
	return nil
}

type FakeSockConnection struct{}

func (f *FakeSockConnection) Id() string {
	return "test1"
}

func (f *FakeSockConnection) Request() *http.Request {
	return &http.Request{}
}

func (f *FakeSockConnection) Close() error {
	return nil
}

func (f *FakeSockConnection) NextReader() (engineio.MessageType, io.ReadCloser, error) {
	return engineio.MessageText, &FakeReadCloser{}, nil
}

func (f *FakeSockConnection) NextWriter(messageType engineio.MessageType) (io.WriteCloser, error) {
	return &FakeWriteCloser{}, nil
}

func TestHandler(t *testing.T) {
	//BugFix missed
	//Method: handler.onPacket
	//Reason: missed fallthrough after case _ACK:
	//
	// 	case _ACK:
	//		fallthrough   <---- fixed problem
	//
	Convey("Call ACK handler by ACK id received from client", t, func() {
		saver := &FrameSaver{}
		var handlerCalled bool
		baseHandlerInstance := newBaseHandler("some:event", &FakeBroadcastAdaptor{})
		socketInstance := newSocket(&FakeSockConnection{}, baseHandlerInstance)
		c, _ := newCaller(func() { handlerCalled = true })

		socketInstance.acks[0] = c
		socketInstance.onPacket(newDecoder(saver), &packet{Type: _ACK, Id: 0, Data: "[]", NSP: "/"})

		So(len(socketInstance.acks), ShouldEqual, 0)
		So(handlerCalled, ShouldBeTrue)
	})

	Convey("Call BINARY ACK handler by BINARY ACK id received from client", t, func() {
		saver := &FrameSaver{}
		var handlerCalled bool
		baseHandlerInstance := newBaseHandler("some:event", &FakeBroadcastAdaptor{})
		socketInstance := newSocket(&FakeSockConnection{}, baseHandlerInstance)
		c, _ := newCaller(func() { handlerCalled = true })

		socketInstance.acks[0] = c
		socketInstance.onPacket(newDecoder(saver), &packet{Type: _BINARY_ACK, Id: 0, Data: "[]", NSP: "/"})

		So(len(socketInstance.acks), ShouldEqual, 0)
		So(handlerCalled, ShouldBeTrue)
	})
}
