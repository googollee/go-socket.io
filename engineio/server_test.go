package engineio

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/googollee/go-socket.io/engineio/packet"
	"github.com/googollee/go-socket.io/engineio/session"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

func TestEnginePolling(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	svr := NewServer(nil)
	defer svr.Close()

	httpSvr := httptest.NewServer(svr)
	defer httpSvr.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		should := assert.New(t)
		must := require.New(t)

		conn, err := svr.Accept()
		must.Nil(err)
		defer conn.Close()

		ft, r, err := conn.NextReader()
		must.Nil(err)
		should.Equal(session.TEXT, ft)

		b, err := ioutil.ReadAll(r)
		must.Nil(err)
		should.Equal("hello你好", string(b))

		must.Nil(r.Close())

		w, err := conn.NextWriter(session.BINARY)
		must.Nil(err)

		_, err = w.Write([]byte{1, 2, 3, 4})
		must.Nil(err)
		must.Nil(w.Close())
	}()

	dialer := Dialer{
		Transports: []transport.Transport{polling.Default},
	}
	header := http.Header{}
	header.Set("X-EIO-Test", "client")

	cnt, err := dialer.Dial(httpSvr.URL, header)
	must.Nil(err)

	w, err := cnt.NextWriter(session.TEXT)
	must.Nil(err)

	_, err = w.Write([]byte("hello你好"))
	must.Nil(err)
	must.Nil(w.Close())

	ft, r, err := cnt.NextReader()
	must.Nil(err)
	should.Equal(session.BINARY, ft)

	b, err := ioutil.ReadAll(r)
	must.Nil(err)
	should.Equal([]byte{1, 2, 3, 4}, b)

	must.Nil(r.Close())
	must.Nil(cnt.Close())

	wg.Wait()
}

func TestEngineWebsocket(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	svr := NewServer(nil)
	defer svr.Close()

	httpSvr := httptest.NewServer(svr)
	defer httpSvr.Close()

	svrInfo := ""

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		should := assert.New(t)
		must := require.New(t)

		conn, err := svr.Accept()
		must.Nil(err)
		defer conn.Close()

		should.Equal("client", conn.RemoteHeader().Get("X-EIO-Test"))
		u := conn.URL()
		svrInfo = fmt.Sprintf("%s %s %s %s", conn.ID(), u.RawQuery, conn.RemoteAddr(), conn.LocalAddr())
		u.RawQuery = ""
		should.Equal("/", u.String())

		ft, r, err := conn.NextReader()
		must.Nil(err)

		should.Equal(session.TEXT, ft)

		b, err := ioutil.ReadAll(r)
		must.Nil(err)

		should.Equal("hello你好", string(b))
		err = r.Close()
		must.Nil(err)

		w, err := conn.NextWriter(session.BINARY)
		must.Nil(err)

		_, err = w.Write([]byte{1, 2, 3, 4})
		must.Nil(err)
		must.Nil(w.Close())
	}()

	dialer := Dialer{
		Transports: []transport.Transport{websocket.Default},
	}
	header := http.Header{}
	header.Set("X-EIO-Test", "client")

	cnt, err := dialer.Dial(httpSvr.URL, header)
	must.Nil(err)

	u := strings.Replace(httpSvr.URL, "http", "ws", 1)
	ur := cnt.URL()
	cntInfo := fmt.Sprintf("%s %s %s %s", cnt.ID(), ur.RawQuery, cnt.LocalAddr(), cnt.RemoteAddr())
	ur.RawQuery = ""
	should.Equal(u, ur.String())

	w, err := cnt.NextWriter(session.TEXT)
	must.Nil(err)

	_, err = w.Write([]byte("hello你好"))
	must.Nil(err)

	err = w.Close()
	must.Nil(err)

	ft, r, err := cnt.NextReader()
	must.Nil(err)
	should.Equal(session.BINARY, ft)

	b, err := ioutil.ReadAll(r)
	must.Nil(err)
	should.Equal([]byte{1, 2, 3, 4}, b)

	err = r.Close()
	must.Nil(err)

	must.Nil(cnt.Close())

	wg.Wait()

	should.Equal(cntInfo, svrInfo)
}

func TestEngineUpgrade(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	svr := NewServer(nil)
	defer svr.Close()

	httpSvr := httptest.NewServer(svr)
	defer httpSvr.Close()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		should := assert.New(t)
		must := require.New(t)

		conn, err := svr.Accept()
		must.Nil(err)
		defer conn.Close()

		ft, r, err := conn.NextReader()
		must.Nil(err)
		should.Equal(session.TEXT, ft)

		b, err := ioutil.ReadAll(r)
		must.Nil(err)
		should.Equal("hello你好", string(b))

		must.Nil(r.Close())

		w, err := conn.NextWriter(session.BINARY)
		must.Nil(err)

		_, err = w.Write([]byte{1, 2, 3, 4})
		must.Nil(err)
		must.Nil(w.Close())
	}()

	u, err := url.Parse(httpSvr.URL)
	must.Nil(err)

	query := u.Query()
	query.Set("EIO", "3")
	u.RawQuery = query.Encode()

	p, err := polling.Default.Dial(u, nil)
	must.Nil(err)

	params, err := p.(Opener).Open()
	must.Nil(err)

	pRead := make(chan int, 1)

	go func() {
		should := assert.New(t)
		must := require.New(t)
		pRead <- 1

		ft, pt, r, err := p.NextReader()
		must.Nil(err)

		should.Equal(packet.FrameString, ft)
		should.Equal(packet.NOOP, pt)
		must.Nil(r.Close())

		close(pRead)
	}()

	<-pRead

	upU := *u
	upU.Scheme = "ws"
	query = upU.Query()
	query.Set("sid", params.SID)
	upU.RawQuery = query.Encode()

	ws, err := websocket.Default.Dial(&upU, nil)
	must.Nil(err)

	w, err := ws.NextWriter(packet.FrameString, packet.PING)
	must.Nil(err)

	_, err = w.Write([]byte("probe"))
	must.Nil(err)

	must.Nil(w.Close())

	ft, pt, r, err := ws.NextReader()
	must.Nil(err)

	should.Equal(packet.FrameString, ft)
	should.Equal(packet.PONG, pt)

	b, err := ioutil.ReadAll(r)
	must.Nil(err)

	should.Equal("probe", string(b))

	must.Nil(r.Close())

	w, err = ws.NextWriter(packet.FrameString, packet.UPGRADE)
	must.Nil(err)

	must.Nil(w.Close())

	<-pRead

	must.Nil(p.Close())

	w, err = ws.NextWriter(packet.FrameString, packet.MESSAGE)
	must.Nil(err)

	_, err = w.Write([]byte("hello你好"))
	must.Nil(err)

	must.Nil(w.Close())

	ft, pt, r, err = ws.NextReader()
	must.Nil(err)

	should.Equal(packet.FrameBinary, ft)
	should.Equal(packet.MESSAGE, pt)

	b, err = ioutil.ReadAll(r)
	must.Nil(err)

	must.Nil(r.Close())
	should.Equal([]byte{1, 2, 3, 4}, b)

	wg.Wait()

	must.Nil(ws.Close())
}
