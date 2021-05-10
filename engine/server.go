package engine

import (
	"context"
	"io"
	"net/http"
	"time"
)

type Option struct{}

func DefaultOption() *Option {
	return nil
}

func (o *Option) WithPingInterval(time.Duration) *Option { return o }
func (o *Option) WithPingTimeout(time.Duration) *Option  { return o }
func (o *Option) WithMaxBufferSize(int) *Option          { return o }

type FrameType int

const (
	FrameBinary FrameType = iota
	FrameText
)

type Server struct{}

func New(*Option) *Server {
	return nil
}

// OnXXX should be called before serving HTTP.
func (s *Server) OnOpen(func(Context, *http.Request) error)    {}
func (s *Server) OnUpgrade(func(Context, *http.Request) error) {}
func (s *Server) OnPing(func(Context))                         {}
func (s *Server) OnMessage(func(Context, io.Reader))           {}
func (s *Server) OnError(func(Context, error))                 {}
func (s *Server) OnClosed(func(Context))                       {}

func (s *Server) ServeHTTP(http.ResponseWriter, *http.Request) {}

// Session methods could be called in any goroutine.
type Session interface {
	ID() string
	Transport() string

	Close() error
	Store(key string, value interface{})

	// SendFrame should be called after closed last frame.
	SendFrame(FrameType) (io.WriteCloser, error)
}

type Context interface {
	context.Context
	Session() Session
}

func HTTPError(code int, msg string) error { return nil }
