package base

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOpError struct {
	timeout   bool
	temporary bool
}

func (f fakeOpError) Error() string {
	return "fake error"
}

func (f fakeOpError) Timeout() bool {
	return f.timeout
}

func (f fakeOpError) Temporary() bool {
	return f.temporary
}

func TestOpError(t *testing.T) {
	tests := []struct {
		url       string
		op        string
		err       error
		timeout   bool
		temporary bool
		errString string
	}{
		{"http://domain/abc", "post(write) to", io.EOF, false, false,
			"post(write) to http://domain/abc: EOF",
		},
		{"http://domain/abc", "get(read) from", io.EOF, false, false,
			"get(read) from http://domain/abc: EOF",
		},
		{"http://domain/abc", "post(write) to", fakeOpError{true, false},
			true, false, "post(write) to http://domain/abc: fake error",
		},
		{"http://domain/abc", "get(read) from", fakeOpError{false, true},
			false, true, "get(read) from http://domain/abc: fake error",
		},
	}

	for i, test := range tests {
		err := OpErr(test.url, test.op, test.err)
		e, ok := err.(*OpError)
		require.True(t, ok, fmt.Sprintf(`cast err to OpErr by case: %d`, i))

		assert.Equal(t, test.timeout, e.Timeout())
		assert.Equal(t, test.temporary, e.Temporary())
		assert.Equal(t, test.errString, e.Error())
	}
}

func TestFrameType(t *testing.T) {
	tests := []struct {
		b     byte
		fType FrameType
	}{
		{0, FrameString},
		{1, FrameBinary},
	}

	for _, test := range tests {
		typ := ByteToFrameType(test.b)

		assert.Equal(t, test.fType, typ)
		assert.Equal(t, test.b, typ.Byte())
	}
}

func TestConnParameters(t *testing.T) {
	tests := []struct {
		params ConnParameters
		out    string
	}{
		{
			ConnParameters{
				time.Second * 10,
				time.Second * 5,
				"vCcJKmYQcIf801WDAAAB",
				[]string{"websocket", "polling"},
			},
			`{"pingInterval":10000,"pingTimeout":5000,"sid":"vCcJKmYQcIf801WDAAAB","upgrades":["websocket","polling"]}` + "\n",
		},
	}

	for _, test := range tests {
		buf := bytes.NewBuffer(nil)
		n, err := test.params.WriteTo(buf)
		require.Nil(t, err)

		assert.Equal(t, int64(len(test.out)), n)
		assert.Equal(t, test.out, buf.String())

		conn, err := ReadConnParameters(buf)
		require.Nil(t, err)

		assert.Equal(t, test.params, conn)
	}
}

func BenchmarkConnParameters(b *testing.B) {
	param := ConnParameters{
		time.Second * 10,
		time.Second * 5,
		"vCcJKmYQcIf801WDAAAB",
		[]string{"websocket", "polling"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := param.WriteTo(ioutil.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}
}
