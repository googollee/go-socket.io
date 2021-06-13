package websocket

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/gorilla/websocket"
)

type bufAllocator struct {
	count int64
	pool  sync.Pool
}

func allocator() *bufAllocator {
	ret := &bufAllocator{}
	ret.pool.New = func() interface{} {
		return make([]byte, 1024)
	}

	return ret
}

func (a *bufAllocator) New() []byte {
	atomic.AddInt64(&a.count, 1)
	return a.pool.Get().([]byte)
}

func (a *bufAllocator) Free(b []byte) {
	atomic.AddInt64(&a.count, -1)
	a.pool.Put(b) //nolint:staticcheck // []byte is a type of pointer.
}

func (a *bufAllocator) Check(t *testing.T) {
	count := atomic.LoadInt64(&a.count)
	if count != 0 {
		t.Fatalf("allocator counter is not 0, value: %d", count)
	}
}

type callbackFuncs struct {
	onPingTimeout func(t transport.Transport)
	onFrame       func(t transport.Transport, req *http.Request, ft frame.Type, rd io.Reader) error
	onError       func(t transport.Transport, err error)
}

func (f callbackFuncs) OnPingTimeout(t transport.Transport) {
	if f.onPingTimeout == nil {
		return
	}
	f.onPingTimeout(t)
}

func (f callbackFuncs) OnFrame(t transport.Transport, req *http.Request, ft frame.Type, rd io.Reader) error {
	if f.onFrame == nil {
		return nil
	}
	return f.onFrame(t, req, ft, rd)
}

func (f callbackFuncs) OnError(t transport.Transport, err error) {
	if f.onError == nil {
		return
	}
	f.onError(t, err)
}

func TestWebsocketName(t *testing.T) {
	alloc := allocator()
	defer alloc.Check(t)

	ws := newWebsocket(time.Second, alloc, callbackFuncs{})

	if diff := cmp.Diff(string(transport.Websocket), ws.Name()); diff != "" {
		t.Errorf("websocket.Name():\n%s", diff)
	}
}

func TestWebsocketOnPingTimeout(t *testing.T) {
	alloc := allocator()
	defer alloc.Check(t)

	callbacks := callbackFuncs{}
	pingWant := time.Second / 3
	i := 0
	callbacks.onPingTimeout = func(t transport.Transport) {
		if i >= 5 {
			if err := t.Close(); err != nil {
				panic(err)
			}
			return
		}
		data := fmt.Sprintf("ping%d", i)
		i++
		wr, err := t.SendFrame(frame.Text)
		if err != nil {
			panic(err)
		}
		if _, err := wr.Write([]byte(data)); err != nil {
			panic(err)
		}
		if err := wr.Close(); err != nil {
			panic(err)
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		ws := newWebsocket(pingWant, alloc, callbacks)
		defer ws.Close()

		if err := ws.PrepareHTTP(w, r); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		ws.ServeHTTP(w, r)
	}

	svr := httptest.NewServer(http.HandlerFunc(handler))
	defer svr.Close()

	url := strings.Replace(svr.URL, "http", "ws", 1)
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial to websocket %s error: %s", url, err)
	}
	defer client.Close()

	wants := []struct {
		messageType int
		data        string
		err         string
	}{
		{websocket.TextMessage, "ping0", ""},
		{websocket.TextMessage, "ping1", ""},
		{websocket.TextMessage, "ping2", ""},
		{websocket.TextMessage, "ping3", ""},
		{websocket.TextMessage, "ping4", ""},
		{-1, "", "websocket: close 1006 (abnormal closure): unexpected EOF"},
	}

	for _, want := range wants {
		begin := time.Now()

		mt, data, err := client.ReadMessage()
		if want, got := want.err, err; !(err == nil && want == "") && want != got.Error() {
			t.Fatalf("client reads messages error, want: %s got: %v", want, got)
			continue
		}
		if want, got := want.messageType, mt; want != got {
			t.Fatalf("client reads messages message type, want: %d got: %d", want, got)
		}
		if want, got := want.data, string(data); want != got {
			t.Fatalf("client reads messages data, want: %s got: %s", want, got)
		}

		dur := time.Since(begin)
		if math.Abs(float64(dur)-float64(pingWant)) > 0.01*float64(time.Second) {
			t.Errorf("want ping duration: %s, got: %s", pingWant, dur)
		}
	}

	client.Close()
}
