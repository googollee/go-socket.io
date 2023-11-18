package polling

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNormalizeMime(t *testing.T) {
	at := assert.New(t)

	tests := []struct {
		mime          string
		supportBinary bool
		ok            bool
	}{
		{"application/octet-stream", true, true},
		{"text/plain; charset=utf-8", false, true},
		{"text/plain;charset=UTF-8", false, true},

		{"text/plain;charset=gbk", false, false},
		{"text/plain charset=U;TF-8", false, false},
		{"text/html", false, false},
	}

	for _, test := range tests {
		isSupportBinary, err := mimeIsSupportBinary(test.mime)
		at.Equal(test.ok, err == nil)

		if err != nil {
			continue
		}

		at.Equal(test.supportBinary, isSupportBinary)
	}
}
