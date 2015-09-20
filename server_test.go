package engineio

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/googollee/go-engine.io/parser"
	. "github.com/smartystreets/goconvey/convey"
)

func TestServer(t *testing.T) {
	Convey("Setup server", t, func() {
		server, err := NewServer(nil)
		So(err, ShouldBeNil)
		server.SetPingInterval(time.Second)
		So(server.config.PingInterval, ShouldEqual, time.Second)
		server.SetPingTimeout(10 * time.Second)
		So(server.config.PingTimeout, ShouldEqual, 10*time.Second)
		f := func(*http.Request) error { return nil }
		server.SetAllowRequest(f)
		So(server.config.AllowRequest, ShouldEqual, f)
		server.SetAllowUpgrades(false)
		So(server.config.AllowUpgrades, ShouldBeFalse)
		server.SetCookie("prefix")
		So(server.config.Cookie, ShouldEqual, "prefix")
		So(server.GetMaxConnection(), ShouldEqual, 1000)
	})

	Convey("Create server", t, func() {

		Convey("Test new id", func() {
			req, err := http.NewRequest("GET", "/", nil)
			So(err, ShouldBeNil)
			id1 := newId(req)
			id2 := newId(req)
			So(id1, ShouldNotEqual, id2)
		})

	})

	Convey("Max connections", t, func() {
		server, _ := NewServer(nil)
		server.SetMaxConnection(1)

		go func() {
			for i := 0; i < 3; i++ {
				server.Accept()
			}
		}()

		req1 := newOpenReq()
		res1 := httptest.NewRecorder()
		server.ServeHTTP(res1, req1)
		So(res1.Code, ShouldEqual, 200)

		req2 := newOpenReq()
		res2 := httptest.NewRecorder()
		server.ServeHTTP(res2, req2)
		So(res2.Code, ShouldEqual, 503)
		So(strings.TrimSpace(string(res2.Body.Bytes())), ShouldEqual, "too many connections")

		server.onClose(extractSid(res1.Body))

		req3 := newOpenReq()
		res3 := httptest.NewRecorder()
		server.ServeHTTP(res3, req3)
		So(res3.Code, ShouldEqual, 200)

	})
}

func newOpenReq() *http.Request {
	openReq, _ := http.NewRequest("GET", "/", bytes.NewBuffer([]byte{}))
	q := openReq.URL.Query()
	q.Set("transport", "polling")
	openReq.URL.RawQuery = q.Encode()
	return openReq
}

func extractSid(body io.Reader) string {
	payload := parser.NewPayloadDecoder(body)
	packet, _ := payload.Next()
	openRes := map[string]interface{}{}
	json.NewDecoder(packet).Decode(&openRes)
	return openRes["sid"].(string)
}
