package polling

import (
	"errors"
	"io"
	"testing"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
)

func TestOpError(t *testing.T) {
	at := assert.New(t)
	err := errors.New("error")
	tests := []struct {
		url    string
		op     string
		input  error
		output error
	}{
		{"http://abc", "op", nil, nil},
		{"http://abc", "op", io.EOF, io.EOF},
		{"http://abc", "op", base.OpErr("http://abc", "op", err), base.OpErr("http://abc", "op", err)},
		{"http://abc", "op", err, base.OpErr("http://abc", "op", err)},
	}

	for _, test := range tests {
		out := retError(test.url, test.op, test.input)
		at.Equal(test.output, out)
	}
}

func TestNormalizeMime(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		mime string
		typ  base.FrameType
		ok   bool
	}{
		{"application/octet-stream", base.FrameBinary, true},
		{"text/plain; charset=utf-8", base.FrameString, true},
		{"text/plain;charset=UTF-8", base.FrameString, true},

		{"text/plain;charset=gbk", base.FrameString, false},
		{"text/plain charset=U;TF-8", base.FrameString, false},
		{"text/html", base.FrameString, false},
	}

	for _, test := range tests {
		typ, err := normalizeMime(test.mime)
		at.Equal(test.ok, err == nil)
		if err != nil {
			continue
		}
		at.Equal(test.typ, typ)
	}
}
