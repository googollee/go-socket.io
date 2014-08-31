package websocket

import (
	"encoding/hex"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/googollee/go-engine.io/message"
	"github.com/googollee/go-engine.io/parser"
	. "github.com/smartystreets/goconvey/convey"
)

func TestWebsocket(t *testing.T) {

	Convey("Creater", t, func() {
		So(Creater.Name, ShouldEqual, "websocket")
		So(Creater.Server, ShouldEqual, NewServer)
		So(Creater.Client, ShouldEqual, NewClient)
	})

	Convey("Normal work, server part", t, func() {
		sync := make(chan int)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f := newFakeCallback()
			s, err := NewServer(w, r, f)
			So(err, ShouldBeNil)
			defer s.Close()

			{
				req, err := http.NewRequest("GET", "/", nil)
				So(err, ShouldBeNil)
				recoder := httptest.NewRecorder()
				s.ServeHTTP(recoder, req)
				So(recoder.Code, ShouldEqual, http.StatusBadRequest)
			}

			{
				w, err := s.NextWriter(message.MessageText, parser.OPEN)
				So(err, ShouldBeNil)
				err = w.Close()
				So(err, ShouldBeNil)
			}

			{
				<-f.onPacket
				So(f.messageType, ShouldEqual, message.MessageBinary)
				So(f.packetType, ShouldEqual, parser.MESSAGE)
				So(f.err, ShouldBeNil)
				So(string(f.body), ShouldEqual, "测试")
			}

			<-sync
			sync <- 1

			<-sync
			sync <- 1

			{
				w, err := s.NextWriter(message.MessageBinary, parser.NOOP)
				So(err, ShouldBeNil)
				err = w.Close()
				So(err, ShouldBeNil)
			}

			<-sync
			sync <- 1

			{
				<-f.onPacket
				So(f.messageType, ShouldEqual, message.MessageText)
				So(f.packetType, ShouldEqual, parser.MESSAGE)
				So(f.err, ShouldBeNil)
				So(hex.EncodeToString(f.body), ShouldEqual, "e697a5e69cace8aa9e")
			}

			<-sync
			sync <- 1
		}))
		defer server.Close()

		u, _ := url.Parse(server.URL)
		u.Scheme = "ws"
		req, _ := http.NewRequest("GET", u.String(), nil)

		c, _ := NewClient(req)
		defer c.Close()

		{
			w, _ := c.NextWriter(message.MessageBinary, parser.MESSAGE)
			w.Write([]byte("测试"))
			w.Close()
		}

		sync <- 1
		<-sync

		{
			decoder, _ := c.NextReader()
			defer decoder.Close()
			ioutil.ReadAll(decoder)
		}

		sync <- 1
		<-sync

		{
			decoder, _ := c.NextReader()
			defer decoder.Close()
			ioutil.ReadAll(decoder)
		}

		sync <- 1
		<-sync

		{
			w, _ := c.NextWriter(message.MessageText, parser.MESSAGE)
			w.Write([]byte("日本語"))
			w.Close()
		}

		sync <- 1
		<-sync
	})

	Convey("Normal work, client part", t, func() {
		sync := make(chan int)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f := newFakeCallback()
			s, _ := NewServer(w, r, f)
			defer s.Close()

			{
				w, _ := s.NextWriter(message.MessageText, parser.OPEN)
				w.Close()
			}

			{
				<-f.onPacket
			}

			<-sync
			sync <- 1

			<-sync
			sync <- 1

			{
				w, _ := s.NextWriter(message.MessageBinary, parser.NOOP)
				w.Close()
			}

			<-sync
			sync <- 1

			{
				<-f.onPacket
			}

			<-sync
			sync <- 1
		}))
		defer server.Close()

		u, err := url.Parse(server.URL)
		So(err, ShouldBeNil)
		u.Scheme = "ws"
		req, err := http.NewRequest("GET", u.String(), nil)
		So(err, ShouldBeNil)

		c, err := NewClient(req)
		So(err, ShouldBeNil)
		defer c.Close()

		So(c.Response(), ShouldNotBeNil)
		So(c.Response().StatusCode, ShouldEqual, http.StatusSwitchingProtocols)

		{
			w, err := c.NextWriter(message.MessageBinary, parser.MESSAGE)
			So(err, ShouldBeNil)
			_, err = w.Write([]byte("测试"))
			So(err, ShouldBeNil)
			err = w.Close()
			So(err, ShouldBeNil)
		}

		sync <- 1
		<-sync

		{
			decoder, err := c.NextReader()
			So(err, ShouldBeNil)
			defer decoder.Close()
			So(decoder.MessageType(), ShouldEqual, message.MessageText)
			So(decoder.Type(), ShouldEqual, parser.OPEN)
			b, err := ioutil.ReadAll(decoder)
			So(err, ShouldBeNil)
			So(string(b), ShouldEqual, "")
		}

		sync <- 1
		<-sync

		{
			decoder, err := c.NextReader()
			So(err, ShouldBeNil)
			defer decoder.Close()
			So(decoder.MessageType(), ShouldEqual, message.MessageBinary)
			So(decoder.Type(), ShouldEqual, parser.NOOP)
			b, err := ioutil.ReadAll(decoder)
			So(err, ShouldBeNil)
			So(string(b), ShouldEqual, "")
		}

		sync <- 1
		<-sync

		{
			w, err := c.NextWriter(message.MessageText, parser.MESSAGE)
			So(err, ShouldBeNil)
			_, err = w.Write([]byte("日本語"))
			So(err, ShouldBeNil)
			err = w.Close()
			So(err, ShouldBeNil)
		}

		sync <- 1
		<-sync
	})

	Convey("Packet content", t, func() {
		sync := make(chan int)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f := newFakeCallback()
			s, _ := NewServer(w, r, f)
			defer s.Close()

			{
				w, _ := s.NextWriter(message.MessageText, parser.MESSAGE)
				w.Write([]byte("日本語"))
				w.Close()
			}

			sync <- 1
			<-sync
		}))
		defer server.Close()

		u, err := url.Parse(server.URL)
		So(err, ShouldBeNil)
		u.Scheme = "ws"
		req, err := http.NewRequest("GET", u.String(), nil)
		So(err, ShouldBeNil)

		c, err := NewClient(req)
		So(err, ShouldBeNil)
		defer c.Close()

		{
			client := c.(*Client)
			t, r, err := client.conn.NextReader()
			So(err, ShouldBeNil)
			So(t, ShouldEqual, websocket.TextMessage)
			b, err := ioutil.ReadAll(r)
			So(err, ShouldBeNil)
			So(string(b), ShouldEqual, "4日本語")
			So(hex.EncodeToString(b), ShouldEqual, "34e697a5e69cace8aa9e")
		}

		<-sync
		sync <- 1
	})

	Convey("Close", t, func() {
		f := newFakeCallback()
		sync := make(chan int)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, _ := NewServer(w, r, f)
			s.Close()
			s.Close()
			s.Close()
			sync <- 1
		}))
		defer server.Close()

		u, err := url.Parse(server.URL)
		So(err, ShouldBeNil)
		u.Scheme = "ws"
		req, err := http.NewRequest("GET", u.String(), nil)
		So(err, ShouldBeNil)

		c, err := NewClient(req)
		So(err, ShouldBeNil)
		defer c.Close()

		<-sync
		So(f.ClosedCount(), ShouldEqual, 1)
	})

	Convey("Closing by disconnected", t, func() {
		f := newFakeCallback()
		sync := make(chan int)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, _ := NewServer(w, r, f)
			server := s.(*Server)
			server.conn.Close()
			sync <- 1
		}))
		defer server.Close()

		u, err := url.Parse(server.URL)
		So(err, ShouldBeNil)
		u.Scheme = "ws"
		req, err := http.NewRequest("GET", u.String(), nil)
		So(err, ShouldBeNil)

		c, err := NewClient(req)
		So(err, ShouldBeNil)
		defer c.Close()

		<-sync
		So(f.ClosedCount(), ShouldEqual, 1)
	})

}

type fakeCallback struct {
	onPacket    chan bool
	messageType message.MessageType
	packetType  parser.PacketType
	body        []byte
	err         error
	closedCount int
	countLocker sync.Mutex
}

func newFakeCallback() *fakeCallback {
	return &fakeCallback{
		onPacket: make(chan bool),
	}
}

func (f *fakeCallback) OnPacket(r *parser.PacketDecoder) {
	f.packetType = r.Type()
	f.messageType = r.MessageType()
	f.body, f.err = ioutil.ReadAll(r)
	f.onPacket <- true
}

func (f *fakeCallback) OnClose() {
	f.countLocker.Lock()
	defer f.countLocker.Unlock()
	f.closedCount++
}

func (f *fakeCallback) ClosedCount() int {
	f.countLocker.Lock()
	defer f.countLocker.Unlock()
	return f.closedCount
}

func (f *fakeCallback) ServeHTTP(w http.ResponseWriter, r *http.Request) {}
