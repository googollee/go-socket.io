package engineio

import (
	"fmt"
	"net/http"
	"runtime"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSessions(t *testing.T) {
	Convey("Normal test", t, func() {
		ses := newSessions()
		t1 := newFakeTransportCreater(true, "t1")
		transports.Register("t1", true, t1)
		req, err := http.NewRequest("GET", "/", nil)
		tt, err := t1(req)
		So(err, ShouldBeNil)
		id := "abc"
		server := NewServer(DefaultConfig)
		conn, err := newSocket(id, server, tt, req)
		So(err, ShouldBeNil)

		So(len(ses.sessions), ShouldEqual, 0)
		ses.Set(id, conn)
		So(len(ses.sessions), ShouldEqual, 1)
		So(ses.Get(id), ShouldEqual, conn)
		ses.Remove(id)
		So(len(ses.sessions), ShouldEqual, 0)
	})

	Convey("Multithread test", t, func() {
		proc := runtime.GOMAXPROCS(10)
		defer runtime.GOMAXPROCS(proc)

		ses := newSessions()
		pause := make(chan bool)
		cont := make(chan bool)
		n := 100

		for i := 0; i < n; i++ {
			go func(i int) {
				t1 := newFakeTransportCreater(true, "t1")
				transports.Register("t1", true, t1)
				req, _ := http.NewRequest("GET", "/", nil)
				tt, _ := t1(req)
				id := fmt.Sprintf("abc%d", i)
				server := NewServer(DefaultConfig)
				conn, _ := newSocket(id, server, tt, req)

				pause <- true
				<-cont
				ses.Set(id, conn)
				pause <- true
				<-cont
				ses.Remove(id)
				pause <- true
			}(i)
		}

		for i := 0; i < n; i++ {
			<-pause
		}

		for i := 0; i < n; i++ {
			cont <- true
		}
		for i := 0; i < n; i++ {
			<-pause
		}
		So(len(ses.sessions), ShouldEqual, n)

		for i := 0; i < n; i++ {
			cont <- true
		}
		for i := 0; i < n; i++ {
			<-pause
		}

	})

}
