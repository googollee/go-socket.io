package polling

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/googollee/go-socket.io/connection/base"

	"github.com/stretchr/testify/assert"
)

func TestServerJSONP(t *testing.T) {
	at := assert.New(t)
	var scValue atomic.Value

	transport := Default
	conn := make(chan base.Conn, 1)
	handler := func(w http.ResponseWriter, r *http.Request) {
		c := scValue.Load()
		if c == nil {
			co, err := transport.Accept(w, r)
			at.Nil(err)
			scValue.Store(co)
			c = co
			conn <- co
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
