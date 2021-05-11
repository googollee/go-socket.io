package engine

import (
	"context"
	"io"
	"net/http"
)

type Client struct {
	Session
}

func NewClient(...Options) (*Client, error) {
	return nil, nil
}

func (c *Client) Dial(context.Context, *http.Request) error { return nil }
func (c *Client) Close() error                              { return nil }

// OnXXX should be called before serving HTTP.
// The engineio framework processes next messages after OnXXX() done. All callback passing to OnXXX should return ASAP.
func (c *Client) OnUpgrade(func(Context, *http.Request) error) {}
func (c *Client) OnMessage(func(Context, io.Reader))           {}
func (c *Client) OnError(func(Context, error))                 {}
func (c *Client) OnClosed(func(Context))                       {}

// OnPingPong triggers when receiving a ping (in EIO v3) or a pong (in EIO v4) message.
func (c *Client) OnPingPong(func(Context)) {}

func (c *Client) SendFrame(FrameType) (io.WriteCloser, error) {
	return nil, nil
}
