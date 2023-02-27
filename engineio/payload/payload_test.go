package payload

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/packet"
)

func TestPayloadFeedIn(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	p := New(true)
	p.Pause()
	p.Resume()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, test := range tests {
			if len(test.packets) != 1 {
				continue
			}
			r := bytes.NewReader(test.data)
			err := p.FeedIn(r, test.supportBinary)
			must.NoError(err)
		}
	}()

	for _, test := range tests {
		if len(test.packets) != 1 {
			continue
		}
		err := p.SetReadDeadline(time.Now().Add(time.Second / 10))
		require.NoError(t, err)

		ft, pt, r, err := p.NextReader()
		must.NoError(err)
		should.Equal(test.packets[0].ft, ft)
		should.Equal(test.packets[0].pt, pt)

		b, err := ioutil.ReadAll(r)
		must.NoError(err)

		must.Nil(r.Close())

		should.Equal(test.packets[0].data, b)
	}

	err := p.SetReadDeadline(time.Now().Add(time.Second / 10))
	require.NoError(t, err)

	_, _, _, err = p.NextReader()
	should.Equal("read: timeout", err.Error())

	wg.Wait()
}

func TestPayloadFlushOutText(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	var supportBinary bool
	p := New(supportBinary)
	p.Pause()
	p.Resume()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		should := assert.New(t)
		must := require.New(t)

		for _, test := range tests {
			if len(test.packets) != 1 {
				continue
			}
			if test.supportBinary != supportBinary {
				continue
			}
			buf := bytes.NewBuffer(nil)
			err := p.FlushOut(buf)
			must.NoError(err)
			should.Equal(test.data, buf.Bytes())
		}
	}()

	for _, test := range tests {
		if len(test.packets) != 1 {
			continue
		}
		if test.supportBinary != supportBinary {
			continue
		}
		err := p.SetWriteDeadline(time.Now().Add(time.Second / 10))
		require.NoError(t, err)

		w, err := p.NextWriter(test.packets[0].ft, test.packets[0].pt)
		must.NoError(err)

		_, err = w.Write(test.packets[0].data)
		must.NoError(err)
		must.Nil(w.Close())
	}

	err := p.SetWriteDeadline(time.Now().Add(time.Second / 10))
	require.NoError(t, err)

	_, err = p.NextWriter(frame.Binary, packet.OPEN)
	should.Equal("write: timeout", err.Error())

	wg.Wait()
}

func TestPayloadFlushOutBinary(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	var supportBinary bool
	p := New(supportBinary)
	p.Pause()
	p.Resume()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		should := assert.New(t)
		must := require.New(t)

		for _, test := range tests {
			if len(test.packets) != 1 {
				continue
			}
			if test.supportBinary != supportBinary {
				continue
			}

			buf := bytes.NewBuffer(nil)
			err := p.FlushOut(buf)
			must.NoError(err)
			should.Equal(test.data, buf.Bytes())
		}
	}()

	for _, test := range tests {
		if len(test.packets) != 1 {
			continue
		}
		if test.supportBinary != supportBinary {
			continue
		}

		err := p.SetWriteDeadline(time.Now().Add(time.Second / 10))
		must.NoError(err)

		w, err := p.NextWriter(test.packets[0].ft, test.packets[0].pt)
		must.NoError(err)

		_, err = w.Write(test.packets[0].data)
		must.NoError(err)

		err = w.Close()
		must.NoError(err)
	}

	err := p.SetWriteDeadline(time.Now().Add(time.Second / 10))
	must.NoError(err)

	_, err = p.NextWriter(frame.Binary, packet.OPEN)
	should.Equal("write: timeout", err.Error())

	wg.Wait()
}

func TestPayloadWaitNextClose(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	p := New(true)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		_, _, _, err := p.NextReader()
		should.Equal(io.EOF, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		_, err := p.NextWriter(frame.Binary, packet.OPEN)
		should.Equal(io.EOF, err)
	}()

	// let next run
	time.Sleep(time.Second / 10)

	err := p.Close()
	must.NoError(err)

	wg.Wait()

	_, _, _, err = p.NextReader()
	should.Equal(io.EOF, err)

	_, err = p.NextWriter(frame.Binary, packet.OPEN)
	should.Equal(io.EOF, err)

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	should.Equal(io.EOF, err)

	err = p.FlushOut(ioutil.Discard)
	should.Equal(io.EOF, err)
}

func TestPayloadWaitInOutClose(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	p := New(true)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := p.FeedIn(bytes.NewReader([]byte("1:0")), false)
		should.Equal(io.EOF, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := p.FlushOut(ioutil.Discard)
		should.Equal(io.EOF, err)
	}()

	// let next run
	time.Sleep(time.Second / 10)

	must.NoError(p.Close())

	wg.Wait()

	_, _, _, err := p.NextReader()
	should.Equal(io.EOF, err)

	_, err = p.NextWriter(frame.Binary, packet.OPEN)
	should.Equal(io.EOF, err)

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	should.Equal(io.EOF, err)

	err = p.FlushOut(ioutil.Discard)
	should.Equal(io.EOF, err)
}

func TestPayloadPauseClose(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	p := New(true)
	p.Pause()

	err := p.Close()
	must.NoError(err)

	_, _, _, err = p.NextReader()
	should.Equal(io.EOF, err)

	_, err = p.NextWriter(frame.Binary, packet.OPEN)
	should.Equal(io.EOF, err)

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	should.Equal(io.EOF, err)

	err = p.FlushOut(ioutil.Discard)
	should.Equal(io.EOF, err)
}

func TestPayloadNextPause(t *testing.T) {
	should := assert.New(t)

	p := New(true)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		should := assert.New(t)
		must := require.New(t)

		_, _, _, err := p.NextReader()
		op, ok := err.(Error)
		must.True(ok)
		should.True(op.Temporary())
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		should := assert.New(t)
		must := require.New(t)

		_, err := p.NextWriter(frame.Binary, packet.OPEN)
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

	_, err = p.NextWriter(frame.Binary, packet.OPEN)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	b := bytes.NewBuffer(nil)
	err = p.FlushOut(b)
	should.Nil(err)
	should.Equal([]byte{0x0, 0x1, 0xff, '6'}, b.Bytes())
}

func TestPayloadInOutPause(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	p := New(true)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := p.FeedIn(bytes.NewReader([]byte("1:0")), false)
		must.NoError(err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		b := bytes.NewBuffer(nil)
		err := p.FlushOut(b)
		must.NoError(err)

		should.Equal([]byte{0x0, 0x1, 0xff, '6'}, b.Bytes())
	}()

	go func() {
		time.Sleep(time.Second / 10 * 3)

		_, _, r, err := p.NextReader()
		defer func() {
			must.NoError(r.Close())
		}()
		must.NoError(err)

		_, err = io.Copy(ioutil.Discard, r)
		must.NoError(err)
	}()

	//wait other run
	time.Sleep(time.Second / 10)

	start := time.Now()
	p.Pause()
	end := time.Now()
	should.True(end.Sub(start) >= time.Second/10)

	wg.Wait()

	_, _, _, err := p.NextReader()
	op, ok := err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	_, err = p.NextWriter(frame.Binary, packet.OPEN)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	b := bytes.NewBuffer(nil)
	err = p.FlushOut(b)
	must.NoError(err)

	should.Equal([]byte{0x0, 0x1, 0xff, '6'}, b.Bytes())
}

func TestPayloadNextClosePause(t *testing.T) {
	should := assert.New(t)

	p := New(true)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		must := require.New(t)
		err := p.FeedIn(bytes.NewReader([]byte("1:0")), false)
		must.NoError(err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		should := assert.New(t)
		must := require.New(t)

		_, _, r, err := p.NextReader()
		must.NoError(err)

		time.Sleep(time.Second / 2)

		must.Nil(r.Close())

		_, _, _, err = p.NextReader()
		op, ok := err.(Error)
		must.True(ok)
		should.True(op.Temporary())
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		must := require.New(t)
		err := p.FlushOut(ioutil.Discard)
		must.NoError(err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		should := assert.New(t)
		must := require.New(t)

		w, err := p.NextWriter(frame.Binary, packet.OPEN)
		must.NoError(err)

		time.Sleep(time.Second / 2)

		err = w.Close()
		must.NoError(err)

		_, err = p.NextWriter(frame.Binary, packet.OPEN)
		op, ok := err.(Error)
		must.True(ok)
		should.True(op.Temporary())
	}()

	// let next run
	time.Sleep(time.Second / 10)

	begin := time.Now()
	p.Pause()
	end := time.Now()
	should.True(end.Sub(begin) > time.Second/5)

	wg.Wait()

	_, _, _, err := p.NextReader()
	op, ok := err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	_, err = p.NextWriter(frame.Binary, packet.OPEN)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	err = p.FeedIn(bytes.NewReader([]byte("1:0")), false)
	op, ok = err.(Error)
	should.True(ok)
	should.True(op.Temporary())

	b := bytes.NewBuffer(nil)
	err = p.FlushOut(b)
	should.Nil(err)
	should.Equal([]byte{0x0, 0x1, 0xff, '6'}, b.Bytes())
}
