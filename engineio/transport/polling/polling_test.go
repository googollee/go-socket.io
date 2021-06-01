package polling

import (
	"io"
	"io/ioutil"
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
	a.pool.Put(b)
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

type packet struct {
	Type frame.Type
	Body string
}

func TestPollingPost(t *testing.T) {
	tests := []struct {
		input string
		want  []packet
	}{
		{"", []packet{{frame.Text, ""}}},
		{"b", []packet{{frame.Binary, ""}}},
		{"1", []packet{{frame.Text, "1"}}},
		{"bMQ==", []packet{{frame.Binary, "1"}}},
		{"\x1eb", []packet{{frame.Text, ""}, {frame.Binary, ""}}},
		{"\x1eb\x1e", []packet{{frame.Text, ""}, {frame.Binary, ""}, {frame.Text, ""}}},
		{"1\x1ebMQ==", []packet{{frame.Text, "1"}, {frame.Binary, "1"}}},
		{"bMQ==\x1e1", []packet{{frame.Binary, "1"}, {frame.Text, "1"}}},
	}

	alloc := allocator()
	for _, test := range tests {
		var got []packet
		callbacks := callbackFuncs{}
		callbacks.onFrame = func(t transport.Transport, req *http.Request, ft frame.Type, rd io.Reader) error {
			data, _ := ioutil.ReadAll(rd)
			got = append(got, packet{ft, string(data)})
			return nil
		}

		polling := newPolling(100*time.Second, alloc, callbacks)

		req, err := http.NewRequest("POST", "/", strings.NewReader(test.input))
		if err != nil {
			t.Fatalf("case %q: create http request error: %s", test.input, err)
		}
		resp := httptest.NewRecorder()

		polling.ServeHTTP(resp, req)

		if diff := cmp.Diff("ok", resp.Body.String()); diff != "" {
			t.Errorf("case %q: diff:\n%s", test.input, diff)
		}

		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("case %q: diff:\n%s", test.input, diff)
		}

		polling.Close()
	}

	if c := atomic.LoadInt64(&alloc.count); c != 0 {
		t.Errorf("buffer count is not 0, got: %d", c)
	}
}

func TestPollingGet(t *testing.T) {
	tests := []struct {
		input []packet
		want  string
	}{
		{[]packet{{frame.Text, ""}}, ""},
		{[]packet{{frame.Binary, ""}}, "b"},
		{[]packet{{frame.Text, "1"}}, "1"},
		{[]packet{{frame.Binary, ""}, {frame.Text, ""}}, "b\x1e"},
		{[]packet{{frame.Binary, "1"}, {frame.Text, "1"}}, "bMQ==\x1e1"},
		{[]packet{{frame.Text, "1"}, {frame.Binary, "1"}}, "1\x1ebMQ=="},
		{[]packet{{frame.Text, "1234567890"}, {frame.Binary, "1234567890"}}, "1234567890\x1ebMTIzNDU2Nzg5MA=="},
	}

	alloc := allocator()
	for _, test := range tests {
		callbacks := callbackFuncs{}

		polling := newPolling(100*time.Second, alloc, callbacks)

		for _, p := range test.input {
			wr, err := polling.SendFrame(p.Type)
			if err != nil {
				t.Fatalf("case %q, sending %s, SendFrame() error: %s", test.want, p.Body, err)
			}
			n, err := wr.Write([]byte(p.Body))
			if err != nil {
				t.Fatalf("case %q, sending %s, Write() error: %s", test.want, p.Body, err)
			}
			if n != len(p.Body) {
				t.Fatalf("case %q, sending %s, Write() length: want: %d, got: %d", test.want, p.Body, len(p.Body), n)
			}
			if err := wr.Close(); err != nil {
				t.Fatalf("case %q, sending %s, Close() error: %s", test.want, p.Body, err)
			}
		}

		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatalf("case %q: create http request error: %s", test.want, err)
		}
		resp := httptest.NewRecorder()

		polling.ServeHTTP(resp, req)

		if diff := cmp.Diff(test.want, resp.Body.String()); diff != "" {
			t.Errorf("case %q: diff:\n%s", test.want, diff)
		}

		polling.Close()
	}

	if c := atomic.LoadInt64(&alloc.count); c != 0 {
		t.Errorf("buffer count is not 0, got: %d", c)
	}
}

func TestPolllingGetPingTimeout(t *testing.T) {
	alloc := allocator()
	callbacks := callbackFuncs{}
	var pingAt time.Time
	callbacks.onPingTimeout = func(tp transport.Transport) {
		pingAt = time.Now()
		wr, err := tp.SendFrame(frame.Text)
		if err != nil {
			t.Fatalf("send frame in ping error: %s", err)
		}
		if err := wr.Close(); err != nil {
			t.Fatalf("close frame in ping error: %s", err)
		}
	}
	pingInterval := time.Second / 3
	polling := newPolling(pingInterval, alloc, callbacks)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("new request errro: %s", err)
	}
	resp := httptest.NewRecorder()

	start := time.Now()
	polling.ServeHTTP(resp, req)
	dur := time.Now().Sub(start)

	if want, got := http.StatusOK, resp.Code; want != got {
		t.Fatalf("get response code, want: %v, got: %v", want, got)
	}
	if want, got := "", resp.Body.String(); want != got {
		t.Fatalf("get response body, want: %v, got: %v", want, got)
	}

	if dur := pingAt.Sub(start); math.Abs(float64(dur-pingInterval)) < 0.001*float64(time.Second) {
		t.Fatalf("the duration of start -> ping should wait %s, but %s", pingInterval, dur)
	}
	if math.Abs(float64(dur-pingInterval)) < 0.001*float64(time.Second) {
		t.Fatalf("the duration of get response should wait %s, but %s", pingInterval, dur)
	}
}
