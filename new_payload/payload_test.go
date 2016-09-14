package payload

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPayloadFeedIn(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	p := New(true)
	p.Pause()
	p.Resume()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, d := range tests {
			r := bytes.NewReader(d.data)
			p.FeedIn(r, d.supportBinary)
		}
	}()

	for _, d := range tests {
		p.SetReadDeadline(time.Now().Add(time.Second / 10))
		ft, pt, r, err := p.NextReader()
		fmt.Println("read err:", err)
		must.Nil(err)
		should.Equal(d.packet.ft, ft)
		should.Equal(d.packet.pt, pt)
		b, err := ioutil.ReadAll(r)
		must.Nil(err)
		should.Equal(d.packet.data, b)
	}

	p.SetReadDeadline(time.Now().Add(time.Second / 10))
	_, _, _, err := p.NextReader()
	should.Equal("read: timeout", err.Error())

	wg.Wait()
}

func TestPayloadFlushOutText(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	supportBinary := false
	p := New(supportBinary)
	p.Pause()
	p.Resume()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		should := assert.New(t)
		must := require.New(t)
		defer wg.Done()
		for _, d := range tests {
			if d.supportBinary != supportBinary {
				continue
			}
			buf := bytes.NewBuffer(nil)
			err := p.FlushOut(buf)
			must.Nil(err)
			should.Equal(d.data, buf.Bytes())
		}
	}()

	for _, d := range tests {
		if d.supportBinary != supportBinary {
			continue
		}
		p.SetWriteDeadline(time.Now().Add(time.Second / 10))
		w, err := p.NextWriter(d.packet.ft, d.packet.pt)
		fmt.Println("write err:", err)
		must.Nil(err)
		_, err = w.Write(d.packet.data)
		must.Nil(err)
		err = w.Close()
		must.Nil(err)
	}

	p.SetWriteDeadline(time.Now().Add(time.Second / 10))
	_, err := p.NextWriter(base.FrameBinary, base.OPEN)
	should.Equal("write: timeout", err.Error())

	wg.Wait()
}

func TestPayloadFlushOutBinary(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	supportBinary := true
	p := New(supportBinary)
	p.Pause()
	p.Resume()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		should := assert.New(t)
		must := require.New(t)
		defer wg.Done()
		for _, d := range tests {
			if d.supportBinary != supportBinary {
				continue
			}
			buf := bytes.NewBuffer(nil)
			err := p.FlushOut(buf)
			must.Nil(err)
			should.Equal(d.data, buf.Bytes())
		}
	}()

	for _, d := range tests {
		if d.supportBinary != supportBinary {
			continue
		}
		p.SetWriteDeadline(time.Now().Add(time.Second / 10))
		w, err := p.NextWriter(d.packet.ft, d.packet.pt)
		must.Nil(err)
		_, err = w.Write(d.packet.data)
		must.Nil(err)
		err = w.Close()
		must.Nil(err)
	}

	p.SetWriteDeadline(time.Now().Add(time.Second / 10))
	_, err := p.NextWriter(base.FrameBinary, base.OPEN)
	should.Equal("write: timeout", err.Error())

	wg.Wait()
}

func TestPayloadWaitNextClose(t *testing.T) {
	should := assert.New(t)

	p := New(true)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		should := assert.New(t)
		defer wg.Done()
		_, _, _, err := p.NextReader()
		should.Equal(io.EOF, err)
	}()

	wg.Add(1)
	go func() {
		should := assert.New(t)
		defer wg.Done()
		_, err := p.NextWriter(base.FrameBinary, base.OPEN)
		should.Equal(io.EOF, err)
	}()

	// let next run
	time.Sleep(time.Second / 10)
	err := p.Close()
	should.Nil(err)

	wg.Wait()

	_, _, _, err = p.NextReader()
	should.Equal(io.EOF, err)
	_, err = p.NextWriter(base.FrameBinary, base.OPEN)
	should.Equal(io.EOF, err)

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	should.Equal(io.EOF, err)
	err = p.FlushOut(ioutil.Discard)
	should.Equal(io.EOF, err)
}

func TestPayloadWaitInOutClose(t *testing.T) {
	should := assert.New(t)

	p := New(true)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		should := assert.New(t)
		defer wg.Done()
		err := p.FeedIn(bytes.NewReader([]byte("1:0")), false)
		should.Equal(io.EOF, err)
	}()

	wg.Add(1)
	go func() {
		should := assert.New(t)
		defer wg.Done()
		err := p.FlushOut(ioutil.Discard)
		should.Equal(io.EOF, err)
	}()

	// let next run
	time.Sleep(time.Second / 10)
	err := p.Close()
	should.Nil(err)

	wg.Wait()

	_, _, _, err = p.NextReader()
	should.Equal(io.EOF, err)
	_, err = p.NextWriter(base.FrameBinary, base.OPEN)
	should.Equal(io.EOF, err)

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	should.Equal(io.EOF, err)
	err = p.FlushOut(ioutil.Discard)
	should.Equal(io.EOF, err)
}

func TestPayloadPauseClose(t *testing.T) {
	should := assert.New(t)

	p := New(true)
	p.Pause()

	err := p.Close()
	should.Nil(err)

	_, _, _, err = p.NextReader()
	should.Equal(io.EOF, err)
	_, err = p.NextWriter(base.FrameBinary, base.OPEN)
	should.Equal(io.EOF, err)

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	should.Equal(io.EOF, err)
	err = p.FlushOut(ioutil.Discard)
	should.Equal(io.EOF, err)
}

func TestPayloadNextPause(t *testing.T) {
	should := assert.New(t)

	p := New(true)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		should := assert.New(t)
		must := require.New(t)
		defer wg.Done()
		_, _, _, err := p.NextReader()
		op, ok := err.(Error)
		must.True(ok)
		should.True(op.Temporary())
	}()

	wg.Add(1)
	go func() {
		should := assert.New(t)
		must := require.New(t)
		defer wg.Done()
		_, err := p.NextWriter(base.FrameBinary, base.OPEN)
		op, ok := err.(Error)
		must.True(ok)
		should.True(op.Temporary())
	}()

	// let next run
	time.Sleep(time.Second / 10)
	p.Pause()

	wg.Wait()

	_, _, _, err := p.NextReader()
	op, ok := err.(Error)
	should.True(ok)
	should.True(op.Temporary())
	_, err = p.NextWriter(base.FrameBinary, base.OPEN)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())
	err = p.FlushOut(ioutil.Discard)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())
}

func TestPayloadInOutPause(t *testing.T) {
	should := assert.New(t)

	p := New(true)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		should := assert.New(t)
		must := require.New(t)
		defer wg.Done()
		err := p.FeedIn(bytes.NewReader([]byte("1:0")), false)
		op, ok := err.(Error)
		must.True(ok)
		should.True(op.Temporary())
	}()

	wg.Add(1)
	go func() {
		should := assert.New(t)
		must := require.New(t)
		defer wg.Done()
		err := p.FlushOut(ioutil.Discard)
		op, ok := err.(Error)
		must.True(ok)
		should.True(op.Temporary())
	}()

	// let next run
	time.Sleep(time.Second / 10)
	p.Pause()

	wg.Wait()

	_, _, _, err := p.NextReader()
	op, ok := err.(Error)
	should.True(ok)
	should.True(op.Temporary())
	_, err = p.NextWriter(base.FrameBinary, base.OPEN)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())
	err = p.FlushOut(ioutil.Discard)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())
}

func TestPayloadNextClosePause(t *testing.T) {
	should := assert.New(t)

	p := New(true)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		must := require.New(t)
		defer wg.Done()
		err := p.FeedIn(bytes.NewReader([]byte("1:0")), false)
		must.Nil(err)
	}()

	wg.Add(1)
	go func() {
		should := assert.New(t)
		must := require.New(t)
		defer wg.Done()
		_, _, _, err := p.NextReader()
		must.Nil(err)
		time.Sleep(time.Second / 2)

		_, _, _, err = p.NextReader()
		op, ok := err.(Error)
		must.True(ok)
		should.True(op.Temporary())
	}()

	wg.Add(1)
	go func() {
		must := require.New(t)
		defer wg.Done()
		err := p.FlushOut(ioutil.Discard)
		must.Nil(err)
	}()

	wg.Add(1)
	go func() {
		should := assert.New(t)
		must := require.New(t)
		defer wg.Done()
		w, err := p.NextWriter(base.FrameBinary, base.OPEN)
		must.Nil(err)
		time.Sleep(time.Second / 2)
		err = w.Close()
		must.Nil(err)

		w, err = p.NextWriter(base.FrameBinary, base.OPEN)
		op, ok := err.(Error)
		must.True(ok)
		should.True(op.Temporary())
	}()

	// let next run
	time.Sleep(time.Second / 10)
	begin := time.Now()
	fmt.Println("pausing")
	p.Pause()
	end := time.Now()
	should.True(end.Sub(begin) > time.Second/5)

	wg.Wait()

	_, _, _, err := p.NextReader()
	op, ok := err.(Error)
	should.True(ok)
	should.True(op.Temporary())
	_, err = p.NextWriter(base.FrameBinary, base.OPEN)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())
	err = p.FlushOut(ioutil.Discard)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())
}
