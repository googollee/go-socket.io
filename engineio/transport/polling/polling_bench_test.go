package polling

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/googollee/go-socket.io/engineio/frame"
)

func BenchmarkPollingPost(b *testing.B) {
	b.StopTimer()

	input := "1234567890\x1ebMTIzNDU2Nzg5MA=="
	want := "ok"

	callbacks := callbackFuncs{}
	alloc := allocator()
	polling := newPolling(100*time.Second, alloc, callbacks)

	reqReader := strings.NewReader(input)
	req, err := http.NewRequest("POST", "/", reqReader)
	if err != nil {
		b.Fatalf("new request error: %s", err)
	}
	resp := httptest.NewRecorder()

	do := func() {
		reqReader.Reset(input)
		resp.Body.Reset()

		polling.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			b.Fatalf("http get response code is not OK: %d", resp.Code)
		}
		if want, got := want, resp.Body.String(); want != got {
			b.Fatalf("get response error, want: %s, got: %s", want, got)
		}
	}

	// prealloc resources
	do()

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		do()
	}

	if err := polling.Close(); err != nil {
		b.Fatalf("polling close error: %s", err)
	}
	if alloc.count != 0 {
		b.Fatalf("alloc count is NOT 0: %d", alloc.count)
	}
}

func BenchmarkPollingGet(b *testing.B) {
	b.StopTimer()

	callbacks := callbackFuncs{}
	alloc := allocator()
	polling := newPolling(100*time.Second, alloc, callbacks)
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		b.Fatalf("new request error: %s", err)
	}
	resp := httptest.NewRecorder()
	data := "1234567890"
	want := "1234567890\x1ebMTIzNDU2Nzg5MA=="

	do := func() {
		resp.Body.Reset()

		for _, t := range []frame.Type{frame.Text, frame.Binary} {
			wr, _ := polling.SendFrame(t)
			if err != nil {
				b.Fatalf("send %s frame error: %s", t, err)
			}
			n, err := wr.Write([]byte(data))
			if err != nil || n != len(data) {
				b.Fatalf("send %s frame data n: %d, error: %s", t, n, err)
			}
			if err := wr.Close(); err != nil {
				b.Fatalf("send %s frame close error: %s", t, err)
			}
		}

		polling.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			b.Fatalf("http get response code is not OK: %d", resp.Code)
		}
		if want, got := want, resp.Body.String(); want != got {
			b.Fatalf("get response error, want: %s, got: %s", want, got)
		}
	}

	// prealloc resources
	do()

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		do()
	}

	if err := polling.Close(); err != nil {
		b.Fatalf("polling close error: %s", err)
	}
	if alloc.count != 0 {
		b.Fatalf("alloc count is NOT 0: %d", alloc.count)
	}
}
