package base

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
