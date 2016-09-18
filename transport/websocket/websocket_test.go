package websocket

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
)

var tests = []struct {
	ft   base.FrameType
	pt   base.PacketType
	data []byte
}{
	{base.FrameString, base.OPEN, []byte{}},
	{base.FrameString, base.MESSAGE, []byte("hello")},
	{base.FrameBinary, base.MESSAGE, []byte{1, 2, 3, 4}},
}

func TestWebsocket(t *testing.T) {
	at := assert.New(t)

	tran := &Transport{}
	at.Equal("websocket", tran.Name())
	conn := make(chan base.Conn, 1)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Eio-Test", "server")
		c, err := tran.Accept(w, r)
		at.Nil(err)
		conn <- c
		c.(http.Handler).ServeHTTP(w, r)
	}
	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	u, err := url.Parse(httpSvr.URL)
	at.Nil(err)
	u.Scheme = "ws"

	header := make(http.Header)
	header.Set("X-Eio-Test", "client")
	cc, err := tran.Dial(u.String(), header)
	at.Nil(err)
	defer cc.Close()

	sc := <-conn
	defer sc.Close()

	at.Equal(u.String(), cc.URL())
	at.Equal("/", sc.URL())
	at.Equal(sc.LocalAddr(), cc.RemoteAddr())
	at.Equal(cc.LocalAddr(), sc.RemoteAddr())
	at.Equal("server", cc.RemoteHeader().Get("X-Eio-Test"))
	at.Equal("client", sc.RemoteHeader().Get("X-Eio-Test"))

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, test := range tests {
			ft, pt, r, err := cc.NextReader()
			at.Nil(err)

			at.Equal(test.ft, ft)
			at.Equal(test.pt, pt)
			b, err := ioutil.ReadAll(r)
			at.Nil(err)
			at.Equal(test.data, b)

			w, err := cc.NextWriter(ft, pt)
			at.Nil(err)
			_, err = w.Write(b)
			at.Nil(err)
			err = w.Close()
			at.Nil(err)
		}
	}()

	for _, test := range tests {
		w, err := sc.NextWriter(test.ft, test.pt)
		at.Nil(err)
		_, err = w.Write(test.data)
		at.Nil(err)
		err = w.Close()
		at.Nil(err)

		ft, pt, r, err := sc.NextReader()
		at.Nil(err)
		at.Equal(test.ft, ft)
		at.Equal(test.pt, pt)
		b, err := ioutil.ReadAll(r)
		at.Nil(err)
		at.Equal(test.data, b)
	}

	wg.Wait()
}
