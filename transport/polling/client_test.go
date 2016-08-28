package polling

import (
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

func TestDialOpen(t *testing.T) {
	cp := base.ConnParameters{
		PingInterval: time.Second,
		PingTimeout:  time.Minute,
		SID:          "abcdefg",
		Upgrades:     []string{"polling"},
	}
	at := assert.New(t)

	var scValue atomic.Value
	transport := New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		c := scValue.Load()
		if c == nil {
			transport.ServeHTTP(w, r)
			return
		}
		c.(http.Handler).ServeHTTP(w, r)
	}
	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		sc := <-transport.ConnChan()
		defer sc.Close()
		scValue.Store(sc)
		w, err := sc.NextWriter(base.FrameBinary, base.OPEN)
		at.Nil(err)
		_, err = cp.WriteTo(w)
		at.Nil(err)
		err = w.Close()
		at.Nil(err)

		ft, pt, r, err := sc.NextReader()
		at.Nil(err)
		at.Equal(base.FrameString, ft)
		at.Equal(base.MESSAGE, pt)
		b, err := ioutil.ReadAll(r)
		at.Nil(err)
		at.Equal("hello", string(b))
	}()

	u, err := url.Parse(httpSvr.URL)
	at.Nil(err)

	dialer := Dialer{}
	connP, err := dialer.Open(u.String(), nil)
	at.Nil(err)
	at.Equal(cp, connP)

	query := u.Query()
	query.Set("sid", connP.SID)
	u.RawQuery = query.Encode()
	cc, err := dialer.Dial(u.String(), nil)
	at.Nil(err)
	defer cc.Close()

	w, err := cc.NextWriter(base.FrameString, base.MESSAGE)
	at.Nil(err)
	_, err = w.Write([]byte("hello"))
	at.Nil(err)
	err = w.Close()
	at.Nil(err)

	wg.Wait()
}
