package socketio_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// OpError is the error type usually returned by functions in the transport
// package.
type OpError struct {
	URL string
	Op  string
	Err error
}

// OpErr makes an *OpError
func OpErr(url, op string, err error) error {
	return &OpError{
		URL: url,
		Op:  op,
		Err: err,
	}
}

func (e *OpError) Error() string {
	return fmt.Sprintf("%s %s: %s", e.Op, e.URL, e.Err.Error())
}

// Timeout returns true if the error is a timeout.
func (e *OpError) Timeout() bool {
	if r, ok := e.Err.(net.Error); ok {
		return r.Timeout()
	}
	return false
}

// Temporary returns true if the error is temporary.
func (e *OpError) Temporary() bool {
	if r, ok := e.Err.(net.Error); ok {
		return r.Temporary()
	}
	return false
}

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

// ConnParameters is connection parameter of server.
type ConnParameters struct {
	PingInterval time.Duration
	PingTimeout  time.Duration
	SID          string
	Upgrades     []string
}

type jsonParameters struct {
	SID          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval"`
	PingTimeout  int      `json:"pingTimeout"`
}

// ReadConnParameters reads ConnParameters from r.
func ReadConnParameters(r io.Reader) (ConnParameters, error) {
	var param jsonParameters
	if err := json.NewDecoder(r).Decode(&param); err != nil {
		return ConnParameters{}, err
	}
	return ConnParameters{
		SID:          param.SID,
		Upgrades:     param.Upgrades,
		PingInterval: time.Duration(param.PingInterval) * time.Millisecond,
		PingTimeout:  time.Duration(param.PingTimeout) * time.Millisecond,
	}, nil
}

type writer struct {
	i int64
	w io.Writer
}

// WriteTo writes to w with json format.
func (p ConnParameters) WriteTo(w io.Writer) (int64, error) {
	arg := jsonParameters{
		SID:          p.SID,
		Upgrades:     p.Upgrades,
		PingInterval: int(p.PingInterval / time.Millisecond),
		PingTimeout:  int(p.PingTimeout / time.Millisecond),
	}
	writer := writer{
		w: w,
	}
	err := json.NewEncoder(&writer).Encode(arg)
	return writer.i, err
}

func (w *writer) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.i += int64(n)
	return n, err
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
