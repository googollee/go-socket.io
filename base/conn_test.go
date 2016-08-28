package base

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	at := assert.New(t)
	tests := []struct {
		url       string
		op        string
		err       error
		timeout   bool
		temporary bool
		errString string
	}{
		{"http://domain/abc", "post(write) to", io.EOF, false, false, "post(write) to http://domain/abc: EOF"},
		{"http://domain/abc", "get(read) from", io.EOF, false, false, "get(read) from http://domain/abc: EOF"},
		{"http://domain/abc", "post(write) to", fakeOpError{true, false}, true, false, "post(write) to http://domain/abc: fake error"},
		{"http://domain/abc", "get(read) from", fakeOpError{false, true}, false, true, "get(read) from http://domain/abc: fake error"},
	}
	for _, test := range tests {
		err := OpErr(test.url, test.op, test.err)
		e, ok := err.(*OpError)
		at.True(ok)
		at.Equal(test.timeout, e.Timeout())
		at.Equal(test.temporary, e.Temporary())
		at.Equal(test.errString, e.Error())
	}
}

func TestFrameType(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		b    byte
		typ  FrameType
		outb byte
	}{
		{0, FrameString, 0},
		{1, FrameBinary, 1},
	}

	for _, test := range tests {
		typ := ByteToFrameType(test.b)
		at.Equal(test.typ, typ)
		b := typ.Byte()
		at.Equal(test.outb, b)
	}
}

func TestConnParameters(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		para ConnParameters
		out  string
	}{
		{
			ConnParameters{
				time.Second * 10,
				time.Second * 5,
				"vCcJKmYQcIf801WDAAAB",
				[]string{"websocket", "polling"},
			},
			"{\"sid\":\"vCcJKmYQcIf801WDAAAB\",\"upgrades\":[\"websocket\",\"polling\"],\"pingInterval\":10000,\"pingTimeout\":5000}\n",
		},
	}
	for _, test := range tests {
		buf := bytes.NewBuffer(nil)
		n, err := test.para.WriteTo(buf)
		at.Nil(err)
		at.Equal(int64(len(test.out)), n)
		at.Equal(test.out, buf.String())

		conn, err := ReadConnParameters(buf)
		at.Nil(err)
		at.Equal(test.para, conn)
	}
}

func BenchmarkConnParameters(b *testing.B) {
	param := ConnParameters{
		time.Second * 10,
		time.Second * 5,
		"vCcJKmYQcIf801WDAAAB",
		[]string{"websocket", "polling"},
	}
	discarder := ioutil.Discard
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		param.WriteTo(discarder)
	}
}
