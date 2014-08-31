package engineio

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConn(t *testing.T) {

	Convey("Test Id", t, func() {
	})

	Convey("Test Request", t, nil)

	Convey("Test NextReader", t, nil)

	Convey("Test NextWriter", t, nil)

	Convey("Test Close", t, func() {

		Convey("Multi-Close", nil)

		Convey("No NextWriter after Close", nil)

		Convey("No OnPacket after Close", nil)

		Convey("Can ServeHTTP after Close", nil)

		Convey("Can NextReader after Close(OnPacket while Close)", nil)

		Convey("Can OnClose after Close", nil)

	})

	Convey("Test OnClose", t, func() {

		Convey("Multi-OnClose", nil)

		Convey("No NextWriter after OnClose", nil)

		Convey("No ServeHTTP after OnClose", nil)

		Convey("No OnPacket after OnClose", nil)

		Convey("Can NextReader after ServeHTTP(OnPacket while OnClose)", nil)

		Convey("Can Close after OnClose", nil)

	})

}
