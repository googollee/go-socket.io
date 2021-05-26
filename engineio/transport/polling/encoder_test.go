package polling

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googollee/go-socket.io/engineio/frame"
)

func TestEncoderWrite(t *testing.T) {
	tests := []struct {
		data   []string
		types  []frame.Type
		output string
	}{
		{[]string{"12345", "6789"}, []frame.Type{frame.Text, frame.Binary}, "12345\x1ebNjc4OQ=="},
	}

	for _, test := range tests {
		// encoder to write all frames while all frames closed
		var buf1 [100]byte
		fullEncoder := newEncoder(20*time.Second, buf1[:])
		// encoder to write all frames while keeps a non-closed frame
		var buf2 [100]byte
		tailEncoder := newEncoder(20*time.Second, buf2[:])

		for i := range test.data {
			for _, encoder := range []*encoder{fullEncoder, tailEncoder} {
				writer, err := encoder.NextFrame(test.types[i])
				if err != nil {
					t.Fatalf("input %q, create frame with data %s error: %s", test.data, test.data[i], err)
				}

				n, err := writer.Write([]byte(test.data[i]))
				if err != nil {
					t.Fatalf("input %q, write frame with data %s error: %s", test.data, test.data[i], err)
				}
				if n != len(test.data[i]) {
					t.Fatalf("input %q, write frame with data %s, length: %d", test.data, test.data[i], n)
				}

				if err := writer.Close(); err != nil {
					t.Fatalf("input %q, close frame with data %s error: %s", test.data, test.data[i], err)
				}
			}
		}

		var output bytes.Buffer
		if err := fullEncoder.WriteFramesTo(&output); err != nil {
			t.Fatalf("input %q, write frames error: %s", test.data, err)
		}
		if diff := cmp.Diff(output.String(), test.output); diff != "" {
			t.Errorf("input %q, diff:\n%s", test.data, diff)
		}
		if err := fullEncoder.Close(); err != nil {
			t.Fatalf("close encoder error: %s", err)
		}

		tailData := "some data"
		writer, err := tailEncoder.NextFrame(frame.Text)
		if err != nil {
			t.Fatalf("write next frame with tailEncoder error: %s", err)
		}
		n, err := writer.Write([]byte(tailData))
		if err != nil {
			t.Fatalf("write data to frame with tailEncoder error: %s", err)
		}
		if n != len(tailData) {
			t.Fatalf("write data to frame, no enough space")
		}

		output.Reset()
		if err := tailEncoder.WriteFramesTo(&output); err != nil {
			t.Fatalf("input %q, write frames error: %s", test.data, err)
		}
		if diff := cmp.Diff(output.String(), test.output); diff != "" {
			t.Errorf("input %q, diff:\n%s", test.data, diff)
		}

		if err := writer.Close(); err != nil {
			t.Fatalf("close frame with tailEncoder error: %s", err)
		}

		output.Reset()
		if err := tailEncoder.WriteFramesTo(&output); err != nil {
			t.Fatalf("input %q, write frames error: %s", test.data, err)
		}
		if diff := cmp.Diff(output.String(), tailData); diff != "" {
			t.Errorf("input %q, diff:\n%s", test.data, diff)
		}
		if err := tailEncoder.Close(); err != nil {
			t.Fatalf("close encoder error: %s", err)
		}
	}
}

func TestEncoderTimeout(t *testing.T) {
	var buf [100]byte

	wantTimeout := time.Second / 10
	encoder := newEncoder(wantTimeout, buf[:])
	start := time.Now()
	defer encoder.Close()

	var output bytes.Buffer
	if want, err := ErrPingTimeout, encoder.WriteFramesTo(&output); err != want {
		t.Fatalf("err want: %s, got: %s", want, err)
	}

	dur := time.Now().Sub(start)
	diff := dur - wantTimeout
	if math.Abs(float64(diff)) >= 0.01*float64(time.Second) {
		t.Fatalf("timeout want: %s, got: %s", wantTimeout, dur)
	}
}

func TestEncoderWriteFrameWhileWait(t *testing.T) {
	var buf [100]byte

	wantWait := time.Second / 10
	encoder := newEncoder(wantWait*2, buf[:])
	defer encoder.Close()

	data := "some data"

	var frameError error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(wantWait)

		writer, err := encoder.NextFrame(frame.Text)
		if err != nil {
			frameError = fmt.Errorf("write next frame error: %w", err)
			return
		}

		n, err := writer.Write([]byte(data))
		if err != nil {
			frameError = fmt.Errorf("write frame error: %w", err)
			return
		}
		if want, got := len(data), n; want != got {
			frameError = fmt.Errorf("write length, want: %d, got: %d", want, got)
			return
		}

		if err := writer.Close(); err != nil {
			frameError = fmt.Errorf("close frame error: %w", err)
			return
		}
	}()

	start := time.Now()
	var output bytes.Buffer
	if err := encoder.WriteFramesTo(&output); err != nil {
		t.Fatalf("write frames to buffer error: %s", err)
	}
	gotWait := time.Now().Sub(start)

	wg.Wait()
	if frameError != nil {
		t.Fatalf("write frames error: %s", frameError)
	}
	if math.Abs(float64(gotWait-wantWait)) >= 0.01*float64(time.Second) {
		t.Fatalf("wait on WriteFramesTo(), want: %s, got: %s", wantWait, gotWait)
	}

	if want, got := data, output.String(); want != got {
		t.Fatalf("output, want: %s, got: %s", want, got)
	}
}

func TestEncoderWriteSeparatorInTextFrame(t *testing.T) {
	var buf [100]byte

	ping := time.Second / 10
	encoder := newEncoder(ping, buf[:])
	defer encoder.Close()

	writer, err := encoder.NextFrame(frame.Text)
	if err != nil {
		t.Fatalf("write text frame error: %s", err)
	}

	n, err := writer.Write([]byte{'a', 'b', separator, 'c', 'd'})
	if want, got := ErrSeparatorInTextFrame, err; want != err {
		t.Fatalf("write with separator want: %s, got: %s", want, got)
	}
	if want, got := 0, n; want != got {
		t.Fatalf("write with separator, lenght want: %d, got: %d", want, got)
	}

	_, err = encoder.NextFrame(frame.Text)
	if want, got := ErrNonCloseFrame, err; want != got {
		t.Fatalf("write next frame error, want: %s, got: %s", want, got)
	}

	// Should able to continue writing.
	data := "some data"
	_, err = writer.Write([]byte(data))
	if err != nil {
		t.Fatalf("writer should be able to continue write, got: %s", err)
	}
	_ = writer.Close()

	var output bytes.Buffer
	_ = encoder.WriteFramesTo(&output)
	if diff := cmp.Diff(data, output.String()); diff != "" {
		t.Errorf("output diff:\n%s", diff)
	}
}

func TestEncoderClose(t *testing.T) {
	var buf [100]byte

	ping := time.Second / 10
	encoder := newEncoder(ping, buf[:])

	if err := encoder.Close(); err != nil {
		t.Fatalf("encoder.Close() error: %s", err)
	}

	if _, err := encoder.NextFrame(frame.Text); err != io.EOF {
		t.Fatalf("encoder.NextFrame() after closing should get io.EOF, got: %s", err)
	}

	var output bytes.Buffer
	if err := encoder.WriteFramesTo(&output); err != io.EOF {
		t.Fatalf("encoder.WriteFramesTo() after closing should get io.EOF, got: %s", err)
	}

	if err := encoder.Close(); err != nil {
		t.Fatalf("encoder.Close() should be called mutliple times, but got: %s", err)
	}
}

func TestEncoderCloseWaitFrame(t *testing.T) {
	var buf [100]byte

	ping := time.Second / 10
	encoder := newEncoder(ping, buf[:])

	wr, err := encoder.NextFrame(frame.Text)
	if err != nil {
		t.Fatalf("encoder.NextFrame() error: %s", err)
	}

	n, err := wr.Write([]byte("1234"))
	if err != nil || n != 4 {
		t.Fatalf("wr.Write() want: (4, nil), got: (%d, %s)", n, err)
	}

	wantWait := time.Second / 10
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(wantWait)
		_ = wr.Close()
	}()

	start := time.Now()
	if err := encoder.Close(); err != nil {
		t.Fatalf("encoder.Close() error: %s", err)
	}
	gotWait := time.Now().Sub(start)

	if math.Abs(float64(gotWait-wantWait)) >= float64(time.Second)*0.01 {
		t.Errorf("close wait time, want: %s, got: %s", wantWait, gotWait)
	}
}
