package polling

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type arrayReader struct {
	data []string
}

func (r *arrayReader) Read(b []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}

	l := len(r.data[0])
	if l > len(b) {
		l = len(b)
	}
	copy(b, r.data[0][:l])

	r.data[0] = r.data[0][l:]
	if len(r.data[0]) == 0 {
		r.data = r.data[1:]
	}

	return l, nil
}

func TestBufReaderRead(t *testing.T) {
	const bufSize = 5
	tests := []struct {
		data []string
		want string
	}{
		{[]string{}, ""},
		{[]string{""}, ""},
		{[]string{"123", "45678", "901234567890"}, "12345678901234567890"},
		{[]string{"12345", "67890", "12345678", "90"}, "12345678901234567890"},
		{[]string{"12345678", "901", "234567890"}, "12345678901234567890"},
	}

	for _, test := range tests {
		rd := &arrayReader{
			data: test.data,
		}

		var buf [bufSize]byte
		bufReader := newBufReader(buf[:], rd)

		got, err := ioutil.ReadAll(bufReader)
		if err != nil {
			t.Fatalf("read error when reading %q: %s", test.data, err)
		}
		if diff := cmp.Diff(string(got), test.want); diff != "" {
			t.Errorf("read %q diff:\n%s", test.data, diff)
		}
	}
}

func TestBufReaderPushBack(t *testing.T) {
	const bufSize = 5
	tests := []struct {
		data     string
		pushback int
		want     string
	}{
		{"1234567890", 0, "67890"},
		{"1234567890", 1, "567890"},
		{"1234567890", 4, "234567890"},
		{"1234567890", 5, "1234567890"},
	}

	for _, test := range tests {
		var buf [bufSize]byte
		bufReader := newBufReader(buf[:], strings.NewReader(test.data))

		_, err := io.ReadFull(bufReader, buf[:])
		if err != nil {
			t.Fatalf("read error when reading %q: %s", test.data, err)
		}

		if err := bufReader.PushBack(test.pushback); err != nil {
			t.Fatalf("PushBack(%d) error when reading %q: %s", test.pushback, test.data, err)
		}

		got, err := ioutil.ReadAll(bufReader)
		if err != nil {
			t.Fatalf("read error when reading %q: %s", test.data, err)
		}
		if diff := cmp.Diff(string(got), test.want); diff != "" {
			t.Errorf("read %q after PushBack(%d) diff:\n%s", test.data, test.pushback, diff)
		}
	}
}

func TestBufReaderFillWithEmtpyReader(t *testing.T) {
	var buf [10]byte
	bufReader := newBufReader(buf[:], strings.NewReader(""))
	if err := bufReader.Fill(); err != nil {
		t.Fatalf("first Fill() with an empty reader should not return an error, got: %s", err)
	}
	if err := bufReader.Fill(); err != io.EOF {
		t.Fatalf("first Fill() with an empty reader should return io.EOF, got: %s", err)
	}
}
