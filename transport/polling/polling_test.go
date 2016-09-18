package polling

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	should := assert.New(t)
	var scValue atomic.Value

	cp := base.ConnParameters{
		PingInterval: time.Second,
		PingTimeout:  time.Minute,
		SID:          "abcdefg",
		Upgrades:     []string{"polling"},
	}
	transport := Default
	should.Equal("polling", transport.Name())
	conn := make(chan base.Conn, 1)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Eio-Test", "server")
		c := scValue.Load()
		if c == nil {
			co, err := transport.Accept(w, r)
			should.Nil(err)
			scValue.Store(co)
			c = co
			conn <- co

			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			buf := bytes.NewBuffer(nil)
			cp.WriteTo(buf)
			fmt.Fprintf(w, "%d", buf.Len()+1)
			w.Write([]byte(":0"))
			w.Write(buf.Bytes())
			return
		}
		c.(http.Handler).ServeHTTP(w, r)
	}
	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	u, err := url.Parse(httpSvr.URL)
	should.Nil(err)

	header := make(http.Header)
	header.Set("X-Eio-Test", "client")
	cc, params, err := transport.Open(u.String(), header)
	should.Nil(err)
	should.Equal(cp, params)
	defer cc.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, test := range tests {
			ft, pt, r, err := cc.NextReader()
			should.Nil(err)

			should.Equal(test.ft, ft)
			should.Equal(test.pt, pt)
			b, err := ioutil.ReadAll(r)
			should.Nil(err)
			should.Equal(test.data, b)
			err = r.Close()
			should.Nil(err)

			w, err := cc.NextWriter(ft, pt)
			should.Nil(err)
			_, err = w.Write(b)
			should.Nil(err)
			err = w.Close()
			should.Nil(err)
		}
	}()

	sc := <-conn
	defer sc.Close()

	for _, test := range tests {
		w, err := sc.NextWriter(test.ft, test.pt)
		should.Nil(err)
		_, err = w.Write(test.data)
		should.Nil(err)
		err = w.Close()
		should.Nil(err)

		ft, pt, r, err := sc.NextReader()
		should.Nil(err)
		should.Equal(test.ft, ft)
		should.Equal(test.pt, pt)
		b, err := ioutil.ReadAll(r)
		should.Nil(err)
		err = r.Close()
		should.Nil(err)
		should.Equal(test.data, b)
	}

	wg.Wait()

	should.Equal(sc.LocalAddr(), cc.RemoteAddr())
	should.Equal(cc.LocalAddr(), "")
	should.NotEqual(sc.RemoteAddr(), "")
	should.Equal("server", cc.RemoteHeader().Get("X-Eio-Test"))
	should.Equal("client", sc.RemoteHeader().Get("X-Eio-Test"))
}

func TestPollingString(t *testing.T) {
	should := assert.New(t)
	var scValue atomic.Value

	cp := base.ConnParameters{
		PingInterval: time.Second,
		PingTimeout:  time.Minute,
		SID:          "abcdefg",
		Upgrades:     []string{"polling"},
	}
	transport := Default
	conn := make(chan base.Conn, 1)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Eio-Test", "server")
		c := scValue.Load()
		if c == nil {
			co, err := transport.Accept(w, r)
			should.Nil(err)
			scValue.Store(co)
			c = co
			conn <- co

			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			buf := bytes.NewBuffer(nil)
			cp.WriteTo(buf)
			fmt.Fprintf(w, "%d", buf.Len()+1)
			w.Write([]byte(":0"))
			w.Write(buf.Bytes())
			return
		}
		c.(http.Handler).ServeHTTP(w, r)
	}
	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	u, err := url.Parse(httpSvr.URL)
	should.Nil(err)

	query := u.Query()
	query.Set("b64", "1")
	u.RawQuery = query.Encode()

	header := make(http.Header)
	header.Set("X-Eio-Test", "client")
	cc, params, err := transport.Open(u.String(), header)
	should.Nil(err)
	should.Equal(cp, params)
	defer cc.Close()

	sc := <-conn
	defer sc.Close()

	should.Equal(sc.LocalAddr(), cc.RemoteAddr())
	should.Equal(cc.LocalAddr(), "")
	should.NotEqual(sc.RemoteAddr(), "")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, test := range tests {
			ft, pt, r, err := cc.NextReader()
			should.Nil(err)

			should.Equal(test.ft, ft)
			should.Equal(test.pt, pt)
			b, err := ioutil.ReadAll(r)
			should.Nil(err)
			err = r.Close()
			should.Nil(err)
			should.Equal(test.data, b)

			w, err := cc.NextWriter(ft, pt)
			should.Nil(err)
			_, err = w.Write(b)
			should.Nil(err)
			err = w.Close()
			should.Nil(err)
		}
	}()

	for _, test := range tests {
		w, err := sc.NextWriter(test.ft, test.pt)
		should.Nil(err)
		_, err = w.Write(test.data)
		should.Nil(err)
		err = w.Close()
		should.Nil(err)

		ft, pt, r, err := sc.NextReader()
		should.Nil(err)
		should.Equal(test.ft, ft)
		should.Equal(test.pt, pt)
		b, err := ioutil.ReadAll(r)
		should.Nil(err)
		err = r.Close()
		should.Nil(err)
		should.Equal(test.data, b)
	}

	wg.Wait()

	should.Equal("server", cc.RemoteHeader().Get("X-Eio-Test"))
	should.Equal("client", sc.RemoteHeader().Get("X-Eio-Test"))
}
