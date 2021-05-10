package engine

import (
	"context"
	"io"
	"net/http"
	"time"
)

type Options func(*Server)

func OptionPingInterval(time.Duration) Options { return nil }
func OptionPingTimeout(time.Duration) Options  { return nil }
func OptionMaxBufferSize(int) Options          { return nil }

type FrameType int

const (
	FrameBinary FrameType = iota
	FrameText
)

type Server struct{}

func New(...Options) *Server {
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
	Get(key string) interface{}

	// SendFrame should be called after closed last frame.
	SendFrame(FrameType) (io.WriteCloser, error)
}

type Context interface {
	context.Context
	Session() Session
}

func HTTPError(code int, msg string) error { return nil }
