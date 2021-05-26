package polling

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googollee/go-socket.io/engineio/frame"
)

func TestDecoder(t *testing.T) {
	tests := []struct {
		input    []string
		want     []string
		wantType []frame.Type
	}{
		{[]string{""}, []string{""}, []frame.Type{frame.Text}},
		{[]string{"b"}, []string{""}, []frame.Type{frame.Binary}},
		{[]string{"12345\x1ebNjc4OQ=="}, []string{"12345", "6789"}, []frame.Type{frame.Text, frame.Binary}},
		{[]string{"bNjc4OQ==\x1e12345"}, []string{"6789", "12345"}, []frame.Type{frame.Binary, frame.Text}},
		{[]string{"1234\x1e", "12345", "\x1e123456"}, []string{"1234", "12345", "123456"}, []frame.Type{frame.Text, frame.Text, frame.Text}},
		{[]string{"123\x1e1234", "56\x1e1", "\x1e12345"}, []string{"123", "123456", "1", "12345"}, []frame.Type{frame.Text, frame.Text, frame.Text, frame.Text}},
		{[]string{"\x1e", "\x1e1", "\x1e\x1e"}, []string{"", "", "1", "", ""}, []frame.Type{frame.Text, frame.Text, frame.Text, frame.Text, frame.Text}},
	}

	for _, test := range tests {
		rd := arrayReader{
			data: test.input,
		}

		var buf [5]byte
		decoder := newDecoder(buf[:], &rd)
		var got []string
		var gotType []frame.Type
		fmt.Println("input:", test.input)

		for {
			ft, rdFrame, err := decoder.NextFrame()
			fmt.Println("next frame error:", err)
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatalf("reading(%q) NextFrame() error: %s", test.input, err)
			}

			b, err := ioutil.ReadAll(rdFrame)
			if err != nil {
				t.Fatalf("reading(%q) frame error: %s", test.input, err)
			}

			got = append(got, string(b))
			gotType = append(gotType, ft)
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

	if _, _, err := decoder.NextFrame(); err != nil {
		t.Fatalf("get next frame error: %s", err)
	}
	ft, rd, err := decoder.NextFrame()
	if err != nil {
		t.Fatalf("get next frame again error: %s", err)
	}

	b, err := ioutil.ReadAll(rd)
	if err != nil {
		t.Fatalf("read frame error: %s", err)
	}
	if want, got := frame.Text, ft; want != got {
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

	ft, rd, err := decoder.NextFrame()
	if err != nil {
		t.Fatalf("get next frame error: %s", err)
	}
	if want, got := frame.Text, ft; want != got {
		t.Fatalf("frame type, want: %v, got: %v", want, got)
	}
	brd, ok := rd.(byteReader)
	if !ok {
		t.Fatalf("text frame reader should have ReadByte(), but not: %v", rd)
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

	ft, rd, err = decoder.NextFrame()
	if err != nil {
		t.Fatalf("get next frame error: %s", err)
	}
	if want, got := frame.Binary, ft; want != got {
		t.Fatalf("frame type, want: %v, got: %v", want, got)
	}
	if _, ok := rd.(byteReader); ok {
		t.Fatalf("binary frame reader should not have ReadByte(), but it have: %v", rd)
	}
}
