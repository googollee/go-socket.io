package engineio

import (
	"bytes"
	"github.com/googollee/go-engine.io/parser"
	"io"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConnIoutil(t *testing.T) {

	Convey("Reader", t, func() {
		Convey("Normal read", func() {
			r := bytes.NewBufferString("\x34\xe6\xb5\x8b\xe8\xaf\x95")
			decoder, err := parser.NewDecoder(r)
			So(err, ShouldBeNil)

			closeChan := make(chan struct{})
			reader := newConnReader(decoder, closeChan)
			b := make([]byte, 1024)
			n, err := reader.Read(b)
			So(err, ShouldBeNil)
			So(string(b[:n]), ShouldEqual, "测试")
			n, err = reader.Read(b)
			So(err, ShouldEqual, io.EOF)

			Convey("Wait close", func() {
				check := make(chan int)
				go func() {
					err := reader.Close()
					So(err, ShouldBeNil)
					check <- 1
				}()
				time.Sleep(time.Second / 10) // wait goroutine start
				select {
				case <-check:
					So("should not run here", ShouldEqual, "")
				default:
				}
				<-closeChan
				time.Sleep(time.Second / 10) // wait goroutine end
				select {
				case <-check:
				default:
					So("should not run here", ShouldEqual, "")
				}

				Convey("Close again", func() {
					err := reader.Close()
					So(err, ShouldBeNil)
				})
			})
		})
	})

}
