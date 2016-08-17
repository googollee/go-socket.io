package payload

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

func TestWriteBinaryLen(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		l      int
		output []byte
	}{
		{0, []byte{0, 0xff}},
		{1, []byte{1, 0xff}},
		{9, []byte{9, 0xff}},
		{10, []byte{1, 0, 0xff}},
		{19, []byte{1, 9, 0xff}},
		{23461, []byte{2, 3, 4, 6, 1, 0xff}},
	}
	for _, test := range tests {
		buf := bytes.NewBuffer(nil)
		w := bufio.NewWriter(buf)
		err := writeBinaryLen(test.l, w)
		w.Flush()
		at.Nil(err)
		at.Equal(test.output, buf.Bytes())
	}

	f := func(l int) bool {
		if l < 0 {
			return true
		}
		buf := bytes.NewBuffer(nil)
		w := bufio.NewWriter(buf)
		writeBinaryLen(l, w)
		w.Flush()
		r := bufio.NewReader(buf)
		out, _ := readBinaryLen(r)
		return out == l
	}
	err := quick.Check(f, nil)
	at.Nil(err)
}

func TestWriteStringLen(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		l      int
		output string
	}{
		{0, "0:"},
		{1, "1:"},
		{9, "9:"},
		{10, "10:"},
		{19, "19:"},
		{23461, "23461:"},
	}
	for _, test := range tests {
		buf := bytes.NewBuffer(nil)
		w := bufio.NewWriter(buf)
		err := writeStringLen(test.l, w)
		w.Flush()
		at.Nil(err)
		at.Equal(test.output, buf.String())
	}

	f := func(l int) bool {
		if l < 0 {
			return true
		}
		buf := bytes.NewBuffer(nil)
		w := bufio.NewWriter(buf)
		writeStringLen(l, w)
		w.Flush()
		r := bufio.NewReader(buf)
		out, _ := readStringLen(r)
		return out == l
	}
	err := quick.Check(f, nil)
	at.Nil(err)
}

func TestReadBytesLen(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		data   []byte
		ok     bool
		output int
	}{
		{[]byte{0xff}, true, 0},
		{[]byte{0, 0xff}, true, 0},
		{[]byte{1, 0xff}, true, 1},
		{[]byte{9, 0xff}, true, 9},
		{[]byte{1, 0, 0xff}, true, 10},
		{[]byte{1, 9, 0xff}, true, 19},
		{[]byte{2, 3, 4, 6, 1, 0xff}, true, 23461},
		{[]byte{2, 3, 4, 6, 1}, false, 0},
		{[]byte{2, 254, 4, 6, 1}, false, 0},
	}
	for _, test := range tests {
		r := bufio.NewReader(bytes.NewReader(test.data))
		l, err := readBinaryLen(r)
		at.Equal(test.ok, err == nil)
		if !test.ok {
			continue
		}
		at.Equal(test.output, l)
	}
}

func TestReadStringLen(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		data   []byte
		ok     bool
		output int
	}{
		{[]byte(":"), true, 0},
		{[]byte("0:"), true, 0},
		{[]byte("1:"), true, 1},
		{[]byte("9:"), true, 9},
		{[]byte("10:"), true, 10},
		{[]byte("19:"), true, 19},
		{[]byte("23461:"), true, 23461},
		{[]byte("23461"), false, 0},
		{[]byte("234ab"), false, 0},
	}
	for _, test := range tests {
		r := bufio.NewReader(bytes.NewReader(test.data))
		l, err := readStringLen(r)
		at.Equal(test.ok, err == nil)
		if !test.ok {
			continue
		}
		at.Equal(test.output, l)
	}
}

func BenchmarkWriteStringLen(b *testing.B) {
	w := bufio.NewWriter(ioutil.Discard)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writeStringLen(23461, w)
	}
}

func BenchmarkWriteBinaryLen(b *testing.B) {
	w := bufio.NewWriter(ioutil.Discard)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writeBinaryLen(23461, w)
	}
}

func BenchmarkReadStringLen(b *testing.B) {
	bs := bytes.Repeat([]byte("23461:"), b.N)
	r := bufio.NewReader(bytes.NewReader(bs))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		readStringLen(r)
	}
}

func BenchmarkReadBinaryLen(b *testing.B) {
	bs := bytes.Repeat([]byte{2, 3, 4, 6, 1, 0xff}, b.N)
	r := bufio.NewReader(bytes.NewReader(bs))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		readBinaryLen(r)
	}
}
