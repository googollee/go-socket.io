package polling

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
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

func TestPollingBinary(t *testing.T) {
	at := assert.New(t)
	var scValue atomic.Value

	transport := New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("handler:", r.Method, r.URL.String())
		w.Header().Set("X-Eio-Test", "server")
		c := scValue.Load()
		if c == nil {
			transport.ServeHTTP(w, r)
			return
		}
		c.(http.Handler).ServeHTTP(w, r)
	}
	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	u, err := url.Parse(httpSvr.URL)
	at.Nil(err)

	dialer := Dialer{}
	header := make(http.Header)
	header.Set("X-Eio-Test", "client")
	cc, err := dialer.Dial(u.String(), header)
	at.Nil(err)
	defer cc.Close()

	sc := <-transport.ConnChan()
	defer sc.Close()
	scValue.Store(sc)

	at.Equal(sc.LocalAddr(), cc.RemoteAddr())
	at.Equal(cc.LocalAddr(), "")
	at.NotEqual(sc.RemoteAddr(), "")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, test := range tests {
			fmt.Println("client next reader begin")
			ft, pt, r, err := cc.NextReader()
			at.Nil(err)
			fmt.Println("client next reader end:", ft, pt)

			at.Equal(test.ft, ft)
			at.Equal(test.pt, pt)
			b, err := ioutil.ReadAll(r)
			at.Nil(err)
			at.Equal(test.data, b)

			fmt.Println("client next writer begin")
			w, err := cc.NextWriter(ft, pt)
			at.Nil(err)
			fmt.Println("client next writer end")
			_, err = w.Write(b)
			at.Nil(err)
			err = w.Close()
			at.Nil(err)
		}
	}()

	for _, test := range tests {
		fmt.Println("server next writer begin")
		w, err := sc.NextWriter(test.ft, test.pt)
		fmt.Println("server next writer end")
		at.Nil(err)
		_, err = w.Write(test.data)
		at.Nil(err)
		err = w.Close()
		at.Nil(err)

		fmt.Println("server next reader begin")
		ft, pt, r, err := sc.NextReader()
		at.Nil(err)
		fmt.Println("server next reader end: ft, pt")
		at.Equal(test.ft, ft)
		at.Equal(test.pt, pt)
		b, err := ioutil.ReadAll(r)
		at.Nil(err)
		at.Equal(test.data, b)
	}

	wg.Wait()

	at.Equal("server", cc.RemoteHeader().Get("X-Eio-Test"))
	at.Equal("client", sc.RemoteHeader().Get("X-Eio-Test"))
}
