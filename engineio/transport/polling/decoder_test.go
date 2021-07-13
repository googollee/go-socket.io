package polling

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googollee/go-socket.io/engineio/frame"
)

func TestDecoder(t *testing.T) {
	tests := []struct {
		input    string
		want     []string
		wantType []frame.Type
	}{
		{"", []string{""}, []frame.Type{frame.Text}},
		{"b", []string{""}, []frame.Type{frame.Binary}},
		{"12345\x1ebNjc4OQ==", []string{"12345", "6789"}, []frame.Type{frame.Text, frame.Binary}},
		{"bNjc4OQ==\x1e12345", []string{"6789", "12345"}, []frame.Type{frame.Binary, frame.Text}},
		{"1234\x1e12345\x1e123456", []string{"1234", "12345", "123456"}, []frame.Type{frame.Text, frame.Text, frame.Text}},
		{"123\x1e123456\x1e1\x1e12345", []string{"123", "123456", "1", "12345"}, []frame.Type{frame.Text, frame.Text, frame.Text, frame.Text}},
		{"\x1e\x1e1\x1e\x1e", []string{"", "", "1", "", ""}, []frame.Type{frame.Text, frame.Text, frame.Text, frame.Text, frame.Text}},
	}

	for _, test := range tests {
		rd := strings.NewReader(test.input)

		var buf [5]byte
		decoder := newDecoder(buf[:], rd)
		var got []string
		var gotType []frame.Type

		for {
			frame, err := decoder.NextFrame()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatalf("reading(%q) NextFrame() error: %s", test.input, err)
			}

			b, err := ioutil.ReadAll(frame.Data)
			if err != nil {
				t.Fatalf("reading(%q) frame error: %s", test.input, err)
			}

			got = append(got, string(b))
			gotType = append(gotType, frame.Type)
		}

		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("reading(%q) frames diff:\n%s", test.input, diff)
		}
		if diff := cmp.Diff(test.wantType, gotType); diff != "" {
			t.Errorf("reading(%q) frames types diff:\n%s", test.input, diff)
		}
	}
}

func TestDecoderIgnoreReading(t *testing.T) {
	data := "1234\x1e5678"
	var buf [50]byte
	decoder := newDecoder(buf[:], strings.NewReader(data))

	if _, err := decoder.NextFrame(); err != nil {
		t.Fatalf("get next frame error: %s", err)
	}
	f, err := decoder.NextFrame()
	if err != nil {
		t.Fatalf("get next frame again error: %s", err)
	}

	b, err := ioutil.ReadAll(f.Data)
	if err != nil {
		t.Fatalf("read frame error: %s", err)
	}
	if want, got := frame.Text, f.Type; want != got {
		t.Fatalf("frame type, want: %v, got: %v", want, got)
	}
	if want, got := "5678", string(b); want != got {
		t.Fatalf("frame data, want: %s, got: %s", want, got)
	}
}

type byteReader interface {
	ReadByte() (byte, error)
}

func TestDecoderTextWithReadByte(t *testing.T) {
	data := "1234\x1ebNjc4OQ=="
	var buf [50]byte
	decoder := newDecoder(buf[:], strings.NewReader(data))

	f, err := decoder.NextFrame()
	if err != nil {
		t.Fatalf("get next frame error: %s", err)
	}
	if want, got := frame.Text, f.Type; want != got {
		t.Fatalf("frame type, want: %v, got: %v", want, got)
	}
	brd, ok := f.Data.(byteReader)
	if !ok {
		t.Fatalf("text frame reader should have ReadByte(), but not: %v", f.Data)
	}
	for _, want := range []byte("1234") {
		got, err := brd.ReadByte()
		if err != nil {
			t.Fatalf("read text error: %s", err)
		}
		if want != got {
			t.Fatalf("read text, want: %x, got: %x", want, got)
		}
	}
	bt, err := brd.ReadByte()
	if err != io.EOF {
		t.Fatalf("read text should return io.EOF, but got: byte(%x), error(%s)", bt, err)
	}

	f, err = decoder.NextFrame()
	if err != nil {
		t.Fatalf("get next frame error: %s", err)
	}
	if want, got := frame.Binary, f.Type; want != got {
		t.Fatalf("frame type, want: %v, got: %v", want, got)
	}
	if _, ok := f.Data.(byteReader); ok {
		t.Fatalf("binary frame reader should not have ReadByte(), but it have: %v", f.Data)
	}
}
