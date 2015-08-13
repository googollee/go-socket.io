package engineio

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestServerSessions(t *testing.T) {
	Convey("Server sessions", t, func() {
		sessions := newServerSessions()

		So(sessions.Count(), ShouldEqual, 0)
		So(sessions.Get("a"), ShouldBeNil)

		sessions.Set("b", new(serverConn))
		So(sessions.Get("b"), ShouldNotBeNil)
		So(sessions.Count(), ShouldEqual, 1)

		So(sessions.Get("a"), ShouldBeNil)

		sessions.Set("c", new(serverConn))
		So(sessions.Get("c"), ShouldNotBeNil)
		So(sessions.Count(), ShouldEqual, 2)

		sessions.Remove("b")
		So(sessions.Get("b"), ShouldBeNil)
		So(sessions.Count(), ShouldEqual, 1)
	})
}
