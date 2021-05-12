package engine

import (
	"context"
	"io"
	"net/http"
	"time"
)

type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
)

type Logger interface {
	Errorf(fmt string, v ...interface{})
	Warnf(fmt string, v ...interface{})
	Infof(fmt string, v ...interface{})
	Debugf(fmt string, v ...interface{})
}

type Options func(*Server)

func OptionPingInterval(time.Duration) Options                      { return nil }
func OptionPingTimeout(time.Duration) Options                       { return nil }
func OptionMaxBufferSize(int) Options                               { return nil }
func OptionLogLevel(level LogLevel) Options                         { return nil }
func OptionLogger(logger Logger) Options                            { return nil }
func OptionTransports(initial string, upgradings ...string) Options { return nil }
func OptionJSONP(padding int) Options                               { return nil }

type FrameType int

const (
	FrameBinary FrameType = iota
	FrameText
)

type PacketType int

const (
	PacketOpen PacketType = iota
	PacketClose
	PacketPing
	PacketPong
	PacketMessage
	PacketUpgrade
	PacketNoop
)

type Server struct{}

func New(...Options) (*Server, error) {
	return nil, nil
}

// OnXXX should be called before serving HTTP.
// The engineio framework processes next messages after OnXXX() done. All callback passing to OnXXX should return ASAP.
func (s *Server) OnOpen(func(Context, *http.Request) error) {}
func (s *Server) OnMessage(func(Context, io.Reader))        {}
func (s *Server) OnError(func(Context, error))              {}
func (s *Server) OnClosed(func(Context))                    {}

// OnPacket calls when receiving packets with type ping/pong/upgrade/noop.
func (s *Server) OnPacket(func(Context, PacketType, io.Reader)) {}

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
