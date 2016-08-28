package websocket

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebsocketSetReadDeadline(t *testing.T) {
	at := assert.New(t)

	svr := New(nil)
	handler := func(w http.ResponseWriter, r *http.Request) {
		svr.ServeHTTP(w, r)
	}
	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	u, err := url.Parse(httpSvr.URL)
	at.Nil(err)
	u.Scheme = "ws"

	dialer := Dialer{}
	header := make(http.Header)
	cc, err := dialer.Dial(u.String(), header)
	at.Nil(err)
	defer cc.Close()

	sc := <-svr.ConnChan()
	defer sc.Close()

	cc.SetReadDeadline(time.Now().Add(time.Second / 10))
	start := time.Now()
	_, _, _, err = cc.NextReader()
	timeout, ok := err.(net.Error)
	at.True(ok)
	at.True(timeout.Timeout())
	end := time.Now()
	at.True(end.Sub(start) > time.Second/10)
}
