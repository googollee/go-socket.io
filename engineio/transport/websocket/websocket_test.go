package websocket

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/packet"
	"github.com/googollee/go-socket.io/engineio/transport"
)

var tests = []struct {
	ft   frame.Type
	pt   packet.Type
	data []byte
}{
	{frame.String, packet.OPEN, []byte{}},
	{frame.String, packet.MESSAGE, []byte("hello")},
	{frame.Binary, packet.MESSAGE, []byte{1, 2, 3, 4}},
}

func TestWebsocket(t *testing.T) {
	wsTransport := &Transport{}
	assert.Equal(t, "websocket", wsTransport.Name())

	conn := make(chan transport.Conn, 1)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Eio-Test", "server")
		c, err := wsTransport.Accept(w, r)
		require.NoError(t, err)

		conn <- c
		c.(http.Handler).ServeHTTP(w, r)
	}
	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	u, err := url.Parse(httpSvr.URL)
	require.NoError(t, err)
	u.Scheme = "ws"

	dialU := *u
	header := make(http.Header)
	header.Set("X-Eio-Test", "client")

	cc, err := wsTransport.Dial(&dialU, header)
	require.NoError(t, err)

	defer cc.Close()

	sc := <-conn
	defer sc.Close()

	ccURL := cc.URL()
	query := ccURL.Query()
	assert.NotEmpty(t, query.Get("t"))

	assert.Equal(t, "websocket", query.Get("transport"))
	ccURL.RawQuery = ""

	assert.Equal(t, u.String(), ccURL.String())
	scURL := sc.URL()
	query = scURL.Query()

	assert.NotEmpty(t, query.Get("t"))
	assert.Equal(t, "websocket", query.Get("transport"))

	scURL.RawQuery = ""
	assert.Equal(t, "/", scURL.String())
	assert.Equal(t, sc.LocalAddr(), cc.RemoteAddr())
	assert.Equal(t, cc.LocalAddr(), sc.RemoteAddr())
	assert.Equal(t, "server", cc.RemoteHeader().Get("X-Eio-Test"))
	assert.Equal(t, "client", sc.RemoteHeader().Get("X-Eio-Test"))

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		for _, test := range tests {
			ft, pt, r, err := cc.NextReader()
			require.NoError(t, err)

			assert.Equal(t, test.ft, ft)
			assert.Equal(t, test.pt, pt)

			b, err := ioutil.ReadAll(r)
			require.NoError(t, err)

			err = r.Close()
			require.NoError(t, err)

			assert.Equal(t, test.data, b)

			w, err := cc.NextWriter(ft, pt)
			require.NoError(t, err)

			_, err = w.Write(b)
			require.NoError(t, err)

			err = w.Close()
			require.NoError(t, err)
		}
	}()

	for _, test := range tests {
		w, err := sc.NextWriter(test.ft, test.pt)
		require.NoError(t, err)

		_, err = w.Write(test.data)
		require.NoError(t, err)

		err = w.Close()
		require.NoError(t, err)

		ft, pt, r, err := sc.NextReader()
		require.NoError(t, err)

		assert.Equal(t, test.ft, ft)
		assert.Equal(t, test.pt, pt)

		b, err := ioutil.ReadAll(r)
		require.NoError(t, err)

		err = r.Close()
		require.NoError(t, err)

		assert.Equal(t, test.data, b)
	}

	wg.Wait()
}
