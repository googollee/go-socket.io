package transport

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnParameters(t *testing.T) {
	must := require.New(t)
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
		must.Nil(err)

		at.Equal(int64(len(test.out)), n)
		at.Equal(test.out, buf.String())

		conn, err := ReadConnParameters(buf)
		must.Nil(err)
		at.Equal(test.para, conn)
	}
}

func BenchmarkConnParameters(b *testing.B) {
	must := require.New(b)

	param := ConnParameters{
		time.Second * 10,
		time.Second * 5,
		"vCcJKmYQcIf801WDAAAB",
		[]string{"websocket", "polling"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := param.WriteTo(ioutil.Discard)
		must.Nil(err)
	}
}
