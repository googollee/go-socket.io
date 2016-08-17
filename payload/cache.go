package payload

import (
	"bytes"
	"encoding/base64"
	"io"

	"github.com/googollee/go-engine.io/base"
)

type frameCache struct {
	e *encoder

	ft   base.FrameType
	pt   base.PacketType
	data *bytes.Buffer
	b64  io.WriteCloser
}

const initSize = 1024

func newFrameCache(e *encoder) *frameCache {
	return &frameCache{
		e:    e,
		data: bytes.NewBuffer(nil),
	}
}

func (c *frameCache) Write(p []byte) (int, error) {
	if c.b64 != nil {
		return c.b64.Write(p)
	}
	return c.data.Write(p)
}

func (c *frameCache) Close() error {
	if c.b64 != nil {
		if err := c.b64.Close(); err != nil {
			return err
		}
	}
	return c.e.closeFrame()
}

func (c *frameCache) Reset(b64 bool, ft base.FrameType, pt base.PacketType) {
	c.ft = ft
	c.pt = pt
	c.data.Reset()
	if b64 {
		c.b64 = base64.NewEncoder(base64.StdEncoding, c.data)
	} else {
		c.b64 = nil
	}
}
