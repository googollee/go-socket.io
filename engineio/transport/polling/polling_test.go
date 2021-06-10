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
	defer alloc.Check(t)

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
	defer alloc.Check(t)

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
}

func TestPolllingGetPingTimeout(t *testing.T) {
	alloc := allocator()
	defer alloc.Check(t)
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
	defer polling.Close()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("new request errro: %s", err)
	}
	resp := httptest.NewRecorder()

	start := time.Now()
	polling.ServeHTTP(resp, req)
	dur := time.Since(start)

	if want, got := http.StatusOK, resp.Code; want != got {
		t.Fatalf("get response code, want: %v, got: %v", want, got)
	}
	if want, got := "", resp.Body.String(); want != got {
		t.Fatalf("get response body, want: %v, got: %v", want, got)
	}

	if dur := pingAt.Sub(start); math.Abs(float64(dur-pingInterval)) >= 0.01*float64(time.Second) {
		t.Fatalf("the duration of start -> ping should wait %s, but %s", pingInterval, dur)
	}
	if math.Abs(float64(dur-pingInterval)) >= 0.01*float64(time.Second) {
		t.Fatalf("the duration of get response should wait %s, but %s", pingInterval, dur)
	}
}

func TestPollingGetOverlapped(t *testing.T) {
	alloc := allocator()
	defer alloc.Check(t)
	wait := make(chan int)
	callbacks := callbackFuncs{}
	callbacks.onPingTimeout = func(t transport.Transport) {
		wait <- 1
		time.Sleep(time.Second / 3) // Let the other GET request go.

		// Write something to let this GET request go.
		wr, _ := t.SendFrame(frame.Text)
		_, _ = wr.Write([]byte("ping"))
		_ = wr.Close()
	}
	pingInterval := time.Second / 10
	polling := newPolling(pingInterval, alloc, callbacks)
	defer polling.Close()

	req1, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("new request errro: %s", err)
	}
	resp1 := httptest.NewRecorder()

	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		polling.ServeHTTP(resp1, req1)
	}()

	<-wait

	req2, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("new request error: %s", err)
	}
	resp2 := httptest.NewRecorder()

	polling.ServeHTTP(resp2, req2)

	if resp2.Code != http.StatusBadRequest {
		t.Fatalf("The second GET should response with code: StatusBadRequest, got: %d", resp2.Code)
	}

	wg.Wait()

	if resp1.Code != http.StatusOK {
		t.Fatalf("The first GET should response with code: StatusOK, got: %d", resp1.Code)
	}
}

func TestPollingPostOverlapped(t *testing.T) {
	alloc := allocator()
	defer alloc.Check(t)
	wait := make(chan int)
	callbacks := callbackFuncs{}
	callbacks.onFrame = func(t transport.Transport, req *http.Request, ft frame.Type, rd io.Reader) error {
		wait <- 1
		time.Sleep(time.Second / 3) // Let the other POST request go.
		return nil
	}
	pingInterval := time.Second / 3
	polling := newPolling(pingInterval, alloc, callbacks)
	defer polling.Close()

	data := "12345"
	req1, err := http.NewRequest("POST", "/", strings.NewReader(data))
	if err != nil {
		t.Fatalf("new request errro: %s", err)
	}
	resp1 := httptest.NewRecorder()

	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		polling.ServeHTTP(resp1, req1)
	}()

	<-wait

	req2, err := http.NewRequest("POST", "/", strings.NewReader(data))
	if err != nil {
		t.Fatalf("new request errro: %s", err)
	}
	resp2 := httptest.NewRecorder()

	polling.ServeHTTP(resp2, req2)

	if resp2.Code != http.StatusBadRequest {
		t.Fatalf("The second GET should response with code: StatusBadRequest, got: %d", resp2.Code)
	}

	wg.Wait()

	if resp1.Code != http.StatusOK {
		t.Fatalf("The first GET should response with code: StatusOK, got: %d", resp1.Code)
	}
}

func TestPollingMethods(t *testing.T) {
	tests := []struct {
		method       string
		responseCode int
	}{
		{"GET", http.StatusOK},
		{"POST", http.StatusOK},

		{"HEAD", http.StatusBadRequest},
		{"OPTIONS", http.StatusBadRequest},
		{"PUT", http.StatusBadRequest},
		{"DELETE", http.StatusBadRequest},
		{"CONNECT", http.StatusBadRequest},
	}

	alloc := allocator()
	defer alloc.Check(t)
	callbacks := callbackFuncs{}
	callbacks.onPingTimeout = func(t transport.Transport) {
		wr, _ := t.SendFrame(frame.Text)
		_, _ = wr.Write([]byte("123"))
		_ = wr.Close()
	}
	pingInterval := time.Second / 10
	polling := newPolling(pingInterval, alloc, callbacks)
	defer polling.Close()

	for _, test := range tests {
		req, err := http.NewRequest(test.method, "/", strings.NewReader(""))
		if err != nil {
			t.Fatalf("create method %s request error: %s", test.method, err)
		}
		resp := httptest.NewRecorder()

		polling.ServeHTTP(resp, req)

		if want, got := test.responseCode, resp.Code; want != got {
			t.Errorf("the response of method %s, want: %d, got: %d", test.method, want, got)
		}
	}
}

func TestPollingClose(t *testing.T) {
	wait := make(chan int)
	alloc := allocator()
	defer alloc.Check(t)
	callbacks := callbackFuncs{}
	blocking := time.Second / 4
	callbacks.onFrame = func(t transport.Transport, req *http.Request, ft frame.Type, rd io.Reader) error {
		wait <- 1
		time.Sleep(blocking)
		return nil
	}
	pingInterval := blocking * 10 // Long ping interval to block writing to Get request.
	polling := newPolling(pingInterval, alloc, callbacks)

	var wg sync.WaitGroup
	defer wg.Wait()

	getReq, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("get request error: %s", err)
	}
	getResp := httptest.NewRecorder()
	wg.Add(1)
	go func() {
		defer wg.Done()
		polling.ServeHTTP(getResp, getReq)
	}()

	postReq, err := http.NewRequest("POST", "/", strings.NewReader("1234"))
	if err != nil {
		t.Fatalf("post request error: %s", err)
	}
	postResp := httptest.NewRecorder()
	wg.Add(1)
	go func() {
		defer wg.Done()
		polling.ServeHTTP(postResp, postReq)
	}()

	time.Sleep(blocking / 10) // wait a while to let requests fire.

	<-wait // get
	start := time.Now()
	if err := polling.Close(); err != nil {
		t.Fatalf("close error: %s", err)
	}
	dur := time.Since(start)
	if math.Abs(float64(dur-blocking)) >= 0.01*float64(time.Second) {
		t.Fatalf("wait %s to close, too long", dur)
	}

	wg.Wait()

	if want, got := http.StatusBadRequest, getResp.Code; want != got {
		t.Fatalf("get response when closing, want: %d, got: %d", want, got)
	}
	if want, got := http.StatusOK, postResp.Code; want != got {
		t.Fatalf("post response when closing, want: %d, got: %d", want, got)
	}

	// Close could be called mutiply times.
	if err := polling.Close(); err != nil {
		t.Fatalf("close error: %s", err)
	}

	// Requests after closed.
	getResp = httptest.NewRecorder()
	polling.ServeHTTP(getResp, getReq)
	if want, got := http.StatusBadRequest, getResp.Code; want != got {
		t.Fatalf("get response when closing, want: %d, got: %d", want, got)
	}

	postResp = httptest.NewRecorder()
	polling.ServeHTTP(postResp, postReq)
	if want, got := http.StatusBadRequest, postResp.Code; want != got {
		t.Fatalf("post response when closing, want: %d, got: %d", want, got)
	}
}

func TestPollingOnFrameError(t *testing.T) {
	tests := []struct {
		code int
	}{
		{http.StatusBadRequest},
	}

	alloc := allocator()
	defer alloc.Check(t)
	callbacks := callbackFuncs{}
	pingInterval := time.Second / 4

	for _, test := range tests {
		callbacks.onFrame = func(t transport.Transport, req *http.Request, ft frame.Type, rd io.Reader) error {
			return transport.HTTPErr(io.EOF, test.code)
		}
		polling := newPolling(pingInterval, alloc, callbacks)

		req, err := http.NewRequest("POST", "/", strings.NewReader("1234"))
		if err != nil {
			t.Fatalf("create request for code %d error: %s", test.code, err)
		}
		resp := httptest.NewRecorder()

		polling.ServeHTTP(resp, req)

		if want, got := test.code, resp.Code; want != got {
			t.Fatalf("response code with OnFrame error, want: %d, got: %d", want, got)
		}

		if err := polling.Close(); err != nil {
			t.Fatalf("code %d, close error: %s", test.code, err)
		}
	}
}

func TestPollingName(t *testing.T) {
	alloc := allocator()
	defer alloc.Check(t)
	callbacks := callbackFuncs{}
	pingInterval := time.Second / 4
	polling := newPolling(pingInterval, alloc, callbacks)
	defer polling.Close()

	if want, got := string(transport.Polling), polling.Name(); want != got {
		t.Errorf("polling.Name(), want: %s, got: %s", want, got)
	}
}
