package polling

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
)

func TestServerJSONP(t *testing.T) {
	at := assert.New(t)
	var scValue atomic.Value

	transport := Default
	conn := make(chan base.Conn)
	handler := func(w http.ResponseWriter, r *http.Request) {
		c := scValue.Load()
		if c == nil {
			transport.ServeHTTP(conn, w, r)
			return
		}
		c.(http.Handler).ServeHTTP(w, r)
	}
	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sc := <-conn
		defer sc.Close()
		scValue.Store(sc)

		w, err := sc.NextWriter(base.FrameBinary, base.MESSAGE)
		at.Nil(err)
		_, err = w.Write([]byte("hello"))
		at.Nil(err)
		err = w.Close()
		at.Nil(err)

		w, err = sc.NextWriter(base.FrameString, base.MESSAGE)
		at.Nil(err)
		_, err = w.Write([]byte("world"))
		at.Nil(err)
		err = w.Close()
		at.Nil(err)
	}()

	{
		u := httpSvr.URL + "?j=jsonp_f1"
		resp, err := http.Get(u)
		at.Nil(err)
		defer resp.Body.Close()

		at.Equal("text/javascript; charset=UTF-8", resp.Header.Get("Content-Type"))
		bs, err := ioutil.ReadAll(resp.Body)
		at.Nil(err)
		at.Equal("___eio[jsonp_f1](\"10:b4aGVsbG8=\");", string(bs))
	}
	{
		u := httpSvr.URL + "?j=jsonp_f2"
		resp, err := http.Get(u)
		at.Nil(err)
		defer resp.Body.Close()

		at.Equal("text/javascript; charset=UTF-8", resp.Header.Get("Content-Type"))
		bs, err := ioutil.ReadAll(resp.Body)
		at.Nil(err)
		at.Equal("___eio[jsonp_f2](\"6:4world\");", string(bs))
	}
	wg.Wait()
}

func TestServerSetReadDeadline(t *testing.T) {
	at := assert.New(t)
	var scValue atomic.Value

	transport := Default
	conn := make(chan base.Conn)
	handler := func(w http.ResponseWriter, r *http.Request) {
		c := scValue.Load()
		if c == nil {
			transport.ServeHTTP(conn, w, r)
			return
		}
		c.(http.Handler).ServeHTTP(w, r)
	}
	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sc := <-conn
		defer sc.Close()
		scValue.Store(sc)

		err := sc.SetReadDeadline(time.Now().Add(time.Second / 10))
		at.Nil(err)

		start := time.Now()
		_, _, _, err = sc.NextReader()
		end := time.Now()

		e, ok := err.(net.Error)
		at.True(ok)
		at.True(e.Timeout())
		at.True(end.Sub(start) > time.Second/10)
	}()

	u := httpSvr.URL
	resp, err := http.Get(u)
	at.Nil(err)
	resp.Body.Close()

	wg.Wait()
}

func TestServerSetWriteDeadline(t *testing.T) {
	at := assert.New(t)
	var scValue atomic.Value

	transport := Default
	conn := make(chan base.Conn)
	handler := func(w http.ResponseWriter, r *http.Request) {
		c := scValue.Load()
		if c == nil {
			transport.ServeHTTP(conn, w, r)
			return
		}
		c.(http.Handler).ServeHTTP(w, r)
	}
	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sc := <-conn
		defer sc.Close()
		scValue.Store(sc)

		err := sc.SetWriteDeadline(time.Now().Add(time.Second / 10))
		at.Nil(err)

		start := time.Now()
		w, err := sc.NextWriter(base.FrameBinary, base.MESSAGE)
		at.Nil(err)
		err = w.Close()
		end := time.Now()

		e, ok := err.(net.Error)
		at.True(ok)
		at.True(e.Timeout())
		at.True(end.Sub(start) > time.Second/10)
	}()

	u := httpSvr.URL
	resp, err := http.Post(u, "plain/text", nil)
	at.Nil(err)
	resp.Body.Close()

	wg.Wait()
}
