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

type FrameType int

const (
	FrameBinary FrameType = iota
	FrameText
)

type Server struct{}

func New(*Option) *Server {
	return nil
}

func (s *Server) OnOpen(func(Context, *http.Request) error)    {}
func (s *Server) OnUpgrade(func(Context, *http.Request) error) {}
func (s *Server) OnPing(func(Context))                         {}
func (s *Server) OnMessage(func(Context, io.Reader))           {}
func (s *Server) OnError(func(Context, error))                 {}
func (s *Server) OnClosed(func(Context))                       {}
func (s *Server) ServeHTTP(http.ResponseWriter, *http.Request) {}

type Session interface {
	ID() string
	Store(key string, value interface{})
	Transport() string
	Close() error
}

type Context interface {
	context.Context
	Session() Session
	SendFrame(FrameType) (io.WriteCloser, error)
}

func HTTPError(code int, msg string) error { return nil }
