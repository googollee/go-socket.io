package engineio

import (
	"github.com/googollee/go-engine.io/message"
	"github.com/googollee/go-engine.io/parser"
	"github.com/googollee/go-engine.io/polling"
	"github.com/googollee/go-engine.io/websocket"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

type FakeServer struct {
	config       *config
	transports   transportCreaters
	closed       map[string]int
	closedLocker sync.Mutex
}

func newFakeServer() *FakeServer {
	return &FakeServer{
		config: &config{
			PingTimeout:   time.Second * 2,
			PingInterval:  time.Second * 1,
			AllowUpgrades: true,
		},
		transports: transportCreaters{
			"polling":   polling.Creater,
			"websocket": websocket.Creater,
		},
		closed: make(map[string]int),
	}
}

func (f *FakeServer) Config() config {
	return *f.config
}

func (f *FakeServer) Transports() transportCreaters {
	return f.transports
}

func (f *FakeServer) OnClose(sid string) {
	f.closedLocker.Lock()
	defer f.closedLocker.Unlock()
	f.closed[sid] = f.closed[sid] + 1
}

func TestConn(t *testing.T) {
	Convey("Create conn", t, func() {
		Convey("without transport", func() {
			server := newFakeServer()
			req, err := http.NewRequest("GET", "/", nil)
			So(err, ShouldBeNil)
			resp := httptest.NewRecorder()
			_, err = NewConn("id", resp, req, server)
			So(err, ShouldEqual, InvalidError)
		})

		Convey("with invalid transport", func() {
			server := newFakeServer()
			req, err := http.NewRequest("GET", "/?transport=websocket", nil)
			So(err, ShouldBeNil)
			resp := httptest.NewRecorder()
			_, err = NewConn("id", resp, req, server)
			So(err, ShouldNotBeNil)
		})

		Convey("ok", func() {
			Convey("with polling", func() {
				server := newFakeServer()
				req, err := http.NewRequest("GET", "/?transport=polling", nil)
				So(err, ShouldBeNil)
				resp := httptest.NewRecorder()
				conn, err := NewConn("id", resp, req, server)
				So(err, ShouldBeNil)
				So(conn.Id(), ShouldEqual, "id")
				So(conn.Request(), ShouldEqual, req)
				conn.Close()
			})

			Convey("with websocket", func() {
				server := newFakeServer()
				h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					conn, err := NewConn("id", w, r, server)
					So(err, ShouldBeNil)
					defer conn.Close()

					So(conn.Id(), ShouldEqual, "id")
					So(conn.Request(), ShouldEqual, r)
				}))
				defer h.Close()

				u, _ := url.Parse(h.URL)
				u.Scheme = "ws"
				req, err := http.NewRequest("GET", u.String()+"/?transport=websocket", nil)
				So(err, ShouldBeNil)
				So(req, ShouldNotBeNil)

				c, err := websocket.NewClient(req)
				So(err, ShouldBeNil)
				defer c.Close()
			})

		})
	})

	Convey("Upgrade conn", t, func() {
		Convey("polling to websocket", func() {
			server := newFakeServer()
			id := "id"
			var conn *serverConn

			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if conn == nil {
					var err error
					conn, err = NewConn(id, w, r, server)
					So(err, ShouldBeNil)
				}

				conn.ServeHTTP(w, r)
			}))
			defer h.Close()

			u, err := url.Parse(h.URL)
			So(err, ShouldBeNil)

			req, err := http.NewRequest("GET", u.String()+"/?transport=polling", nil)
			So(err, ShouldBeNil)
			pc, err := polling.NewClient(req)
			So(err, ShouldBeNil)

			decoder, err := pc.NextReader()
			So(err, ShouldBeNil)
			So(pc.Response().StatusCode, ShouldEqual, http.StatusOK)

			So(conn, ShouldNotBeNil)
			So(conn, ShouldImplement, (*Conn)(nil))

			So(decoder.MessageType(), ShouldEqual, message.MessageText)
			So(decoder.Type(), ShouldEqual, parser.PING)

			So(conn.getCurrent(), ShouldNotBeNil)
			So(conn.getUpgrade(), ShouldBeNil)

			u.Scheme = "ws"
			req, err = http.NewRequest("GET", u.String()+"/?transport=websocket", nil)
			So(err, ShouldBeNil)
			wc, err := websocket.NewClient(req)
			So(err, ShouldBeNil)

			So(conn.getCurrent(), ShouldNotBeNil)
			So(conn.getUpgrade(), ShouldNotBeNil)

			encoder, err := wc.NextWriter(message.MessageBinary, parser.PING)
			So(err, ShouldBeNil)
			encoder.Write([]byte("probe"))
			encoder.Close()

			decoder, err = wc.NextReader()
			So(err, ShouldBeNil)
			So(wc.Response().StatusCode, ShouldEqual, http.StatusSwitchingProtocols)

			So(decoder.MessageType(), ShouldEqual, message.MessageText)
			So(decoder.Type(), ShouldEqual, parser.PONG)

			pc.Close()

			encoder, err = wc.NextWriter(message.MessageBinary, parser.UPGRADE)
			So(err, ShouldBeNil)
			encoder.Close()

			decoder, err = wc.NextReader()
			So(err, ShouldBeNil)
			So(pc.Response().StatusCode, ShouldEqual, http.StatusOK)

			So(decoder.MessageType(), ShouldEqual, message.MessageText)
			So(decoder.Type(), ShouldEqual, parser.PING)

			So(conn.getCurrent(), ShouldNotBeNil)
			So(conn.getUpgrade(), ShouldBeNil)

			wc.Close()

			err = conn.Close()
			So(err, ShouldBeNil)

			time.Sleep(time.Second)

			server.closedLocker.Lock()
			So(server.closed[id], ShouldEqual, 1)
			server.closedLocker.Unlock()
		})

	})
}
