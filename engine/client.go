package engine

import (
	"io"
	"net/http"
)

type Client struct {
	Session
}

func NewClient(...Options) (*Client, error) {
	return nil, nil
}

func (c *Client) Open(*http.Request) error { return nil }
func (c *Client) Close() error             { return nil }

// OnXXX should be called before serving HTTP.
// The engineio framework processes next messages after OnXXX() done. All callback passing to OnXXX should return ASAP.
func (c *Client) OnMessage(func(Context, io.Reader)) {}
func (c *Client) OnError(func(Context, error))       {}
func (c *Client) OnClosed(func(Context))             {}

// OnPacket calls when receiving packets with type ping/pong/upgrade/noop.
func (c *Client) OnPacket(func(Context, PacketType, io.Reader)) {}

func (c *Client) SendFrame(FrameType) (io.WriteCloser, error) {
	return nil, nil
}
