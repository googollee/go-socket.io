package polling

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/packet"
	"github.com/googollee/go-socket.io/engineio/transport"
)

func TestDialOpen(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	cp := transport.ConnParameters{
		PingInterval: time.Second,
		PingTimeout:  time.Minute,
		SID:          "abcdefg",
		Upgrades:     []string{"polling"},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		query := r.URL.Query()
		should.NotEmpty(r.URL.Query().Get("t"))
		sid := query.Get("sid")

		if sid == "" {
			buf := bytes.NewBuffer(nil)
			_, err := cp.WriteTo(buf)
			must.NoError(err)

			_, err = fmt.Fprintf(w, "%d", buf.Len()+1)
			must.NoError(err)

			_, err = w.Write([]byte(":0"))
			must.NoError(err)

			_, err = w.Write(buf.Bytes())
			must.NoError(err)

			return
		}

		if r.Method == http.MethodPost {
			must.Equal(cp.SID, sid)

			b, err := ioutil.ReadAll(r.Body)
			must.NoError(err)

			should.Equal("6:4hello", string(b))
		}
	}

	httpSvr := httptest.NewServer(http.HandlerFunc(handler))
	defer httpSvr.Close()

	u, err := url.Parse(httpSvr.URL)
	must.NoError(err)

	query := u.Query()
	query.Set("b64", "1")
	u.RawQuery = query.Encode()

	cc, err := dial(nil, u, nil)
	must.NoError(err)

	defer func() {
		must.NoError(cc.Close())
	}()

	params, err := cc.Open()
	must.NoError(err)

	should.Equal(cp, params)

	ccURL := cc.URL()
	sid := ccURL.Query().Get("sid")

	should.Equal(cp.SID, sid)

	w, err := cc.NextWriter(frame.String, packet.MESSAGE)
	must.NoError(err)

	_, err = w.Write([]byte("hello"))
	must.NoError(err)
	should.Nil(w.Close())
}
