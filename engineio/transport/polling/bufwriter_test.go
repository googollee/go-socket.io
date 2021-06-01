package polling

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBufWriter(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{[]string{""}, ""},
		{[]string{"123", "4"}, ""},
		{[]string{"123", "45\x1e"}, "12345"},
		{[]string{"123", "45\x1e6", "789"}, "12345"},
		{[]string{"123", "45\x1e6", "789", "\x1e"}, "12345\x1e6789"},
		{[]string{"123", "45\x1e6", "789", "\x1e", "1234"}, "12345\x1e6789"},
	}

	var buf [20]byte
	for _, test := range tests {
		wr := newBufWriter(buf[:])

		for _, data := range test.input {
			var err error
			if len(data) == 1 {
				err = wr.WriteByte(byte(data[0]))
			} else {
				_, err = wr.Write([]byte(data))
			}

			if err != nil {
				t.Fatalf("input %q, got error when writing %q: %v", test.input, data, err)
			}
		}

		var buf bytes.Buffer
		if _, err := wr.WriteFinishedFrames(&buf); err != nil {
			t.Fatalf("input %q, write finished frames error: %v", test.input, err)
		}
		if diff := cmp.Diff(buf.String(), test.want); diff != "" {
			t.Errorf("input %q, write finished frames diff:\n%s", test.input, diff)
		}
	}
}
