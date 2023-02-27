package polling

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/packet"
	"github.com/googollee/go-socket.io/engineio/transport"
)

func TestServerJSONP(t *testing.T) {
	must := require.New(t)

	var scValue atomic.Value

	pollingTransport := Default
	conn := make(chan transport.Conn, 1)

	handler := func(w http.ResponseWriter, r *http.Request) {
		c := scValue.Load()
		if c == nil {
			co, err := pollingTransport.Accept(w, r)
			require.NoError(t, err)

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

		defer func() {
			must.NoError(sc.Close())
		}()

		w, err := sc.NextWriter(frame.Binary, packet.MESSAGE)
		require.NoError(t, err)

		_, err = w.Write([]byte("hello"))
		require.NoError(t, err)

		err = w.Close()
		require.NoError(t, err)

		w, err = sc.NextWriter(frame.String, packet.MESSAGE)
		require.NoError(t, err)

		_, err = w.Write([]byte("world"))
		require.NoError(t, err)

		err = w.Close()
		require.NoError(t, err)
	}()

	{
		u := httpSvr.URL + "?j=jsonp_f1"
		resp, err := http.Get(u)
		require.NoError(t, err)

		defer func() {
			err = resp.Body.Close()
			require.NoError(t, err)
		}()

		assert.Equal(t, "text/javascript; charset=UTF-8", resp.Header.Get("Content-Type"))
		bs, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf("___eio[jsonp_f1](\"%s\");", template.JSEscapeString("10:b4aGVsbG8=")), string(bs))
	}
	{
		u := httpSvr.URL + "?j=jsonp_f2"
		resp, err := http.Get(u)
		require.NoError(t, err)

		defer func() {
			err = resp.Body.Close()
			require.NoError(t, err)
		}()

		assert.Equal(t, "text/javascript; charset=UTF-8", resp.Header.Get("Content-Type"))

		bs, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "___eio[jsonp_f2](\"6:4world\");", string(bs))
	}

	wg.Wait()
}
