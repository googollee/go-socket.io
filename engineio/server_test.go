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

	"github.com/googollee/go-socket.io/engineio/frame"
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
	defer func() {
		must.NoError(svr.Close())
	}()

	httpSvr := httptest.NewServer(svr)
	defer httpSvr.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		conn, err := svr.Accept()
		must.NoError(err)
		defer func() {
			must.NoError(conn.Close())
		}()

		ft, r, err := conn.NextReader()
		must.NoError(err)
		should.Equal(session.TEXT, ft)

		b, err := ioutil.ReadAll(r)
		must.NoError(err)
		should.Equal("hello你好", string(b))

		must.Nil(r.Close())

		w, err := conn.NextWriter(session.BINARY)
		must.NoError(err)

		_, err = w.Write([]byte{1, 2, 3, 4})
		must.NoError(err)
		must.Nil(w.Close())
	}()

	dialer := Dialer{
		Transports: []transport.Transport{polling.Default},
	}
	header := http.Header{}
	header.Set("X-EIO-Test", "client")

	cnt, err := dialer.Dial(httpSvr.URL, header)
	must.NoError(err)

	w, err := cnt.NextWriter(session.TEXT)
	must.NoError(err)

	_, err = w.Write([]byte("hello你好"))
	must.NoError(err)
	must.Nil(w.Close())

	ft, r, err := cnt.NextReader()
	must.NoError(err)
	should.Equal(session.BINARY, ft)

	b, err := ioutil.ReadAll(r)
	must.NoError(err)
	should.Equal([]byte{1, 2, 3, 4}, b)

	must.Nil(r.Close())
	must.Nil(cnt.Close())

	wg.Wait()
}

func TestEngineWebsocket(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	svr := NewServer(nil)
	defer func() {
		must.NoError(svr.Close())
	}()

	httpSvr := httptest.NewServer(svr)
	defer httpSvr.Close()

	svrInfo := ""

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		conn, err := svr.Accept()
		must.NoError(err)

		defer func() {
			must.NoError(conn.Close())
		}()

		should.Equal("client", conn.RemoteHeader().Get("X-EIO-Test"))
		u := conn.URL()
		svrInfo = fmt.Sprintf("%s %s %s %s", conn.ID(), u.RawQuery, conn.RemoteAddr(), conn.LocalAddr())
		u.RawQuery = ""
		should.Equal("/", u.String())

		ft, r, err := conn.NextReader()
		must.NoError(err)

		should.Equal(session.TEXT, ft)

		b, err := ioutil.ReadAll(r)
		must.NoError(err)

		should.Equal("hello你好", string(b))
		err = r.Close()
		must.NoError(err)

		w, err := conn.NextWriter(session.BINARY)
		must.NoError(err)

		_, err = w.Write([]byte{1, 2, 3, 4})
		must.NoError(err)
		must.Nil(w.Close())
	}()

	dialer := Dialer{
		Transports: []transport.Transport{websocket.Default},
	}
	header := http.Header{}
	header.Set("X-EIO-Test", "client")

	cnt, err := dialer.Dial(httpSvr.URL, header)
	must.NoError(err)

	u := strings.Replace(httpSvr.URL, "http", "ws", 1)
	ur := cnt.URL()
	cntInfo := fmt.Sprintf("%s %s %s %s", cnt.ID(), ur.RawQuery, cnt.LocalAddr(), cnt.RemoteAddr())
	ur.RawQuery = ""
	should.Equal(u, ur.String())

	w, err := cnt.NextWriter(session.TEXT)
	must.NoError(err)

	_, err = w.Write([]byte("hello你好"))
	must.NoError(err)

	err = w.Close()
	must.NoError(err)

	ft, r, err := cnt.NextReader()
	must.NoError(err)
	should.Equal(session.BINARY, ft)

	b, err := ioutil.ReadAll(r)
	must.NoError(err)
	should.Equal([]byte{1, 2, 3, 4}, b)

	err = r.Close()
	must.NoError(err)

	must.NoError(cnt.Close())

	wg.Wait()

	should.Equal(cntInfo, svrInfo)
}

func TestEngineUpgrade(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	svr := NewServer(nil)
	defer func() {
		must.NoError(svr.Close())
	}()

	httpSvr := httptest.NewServer(svr)
	defer httpSvr.Close()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		conn, err := svr.Accept()
		must.NoError(err)
		defer func() {
			must.NoError(conn.Close())
		}()

		ft, r, err := conn.NextReader()
		must.NoError(err)
		should.Equal(session.TEXT, ft)

		b, err := ioutil.ReadAll(r)
		must.NoError(err)
		should.Equal("hello你好", string(b))

		must.NoError(r.Close())

		w, err := conn.NextWriter(session.BINARY)
		must.NoError(err)

		_, err = w.Write([]byte{1, 2, 3, 4})
		must.NoError(err)
		must.NoError(w.Close())
	}()

	u, err := url.Parse(httpSvr.URL)
	must.NoError(err)

	query := u.Query()
	query.Set("EIO", "3")
	u.RawQuery = query.Encode()

	p, err := polling.Default.Dial(u, nil)
	must.NoError(err)

	params, err := p.(Opener).Open()
	must.NoError(err)

	pRead := make(chan int, 1)

	go func() {
		pRead <- 1

		ft, pt, r, err := p.NextReader()
		must.NoError(err)

		should.Equal(frame.String, ft)
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
	must.NoError(err)

	w, err := ws.NextWriter(frame.String, packet.PING)
	must.NoError(err)

	_, err = w.Write([]byte("probe"))
	must.NoError(err)

	must.NoError(w.Close())

	ft, pt, r, err := ws.NextReader()
	must.NoError(err)

	should.Equal(frame.String, ft)
	should.Equal(packet.PONG, pt)

	b, err := ioutil.ReadAll(r)
	must.NoError(err)

	should.Equal("probe", string(b))

	must.NoError(r.Close())

	w, err = ws.NextWriter(frame.String, packet.UPGRADE)
	must.NoError(err)

	must.NoError(w.Close())

	<-pRead

	must.Nil(p.Close())

	w, err = ws.NextWriter(frame.String, packet.MESSAGE)
	must.NoError(err)

	_, err = w.Write([]byte("hello你好"))
	must.NoError(err)

	must.Nil(w.Close())

	ft, pt, r, err = ws.NextReader()
	must.NoError(err)

	should.Equal(frame.Binary, ft)
	should.Equal(packet.MESSAGE, pt)

	b, err = ioutil.ReadAll(r)
	must.NoError(err)

	must.NoError(r.Close())
	should.Equal([]byte{1, 2, 3, 4}, b)

	wg.Wait()

	must.NoError(ws.Close())
}
