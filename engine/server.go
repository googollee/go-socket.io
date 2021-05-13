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
func (s *Server) OnOpen(func(*Context, *http.Request) error) {}
func (s *Server) OnMessage(func(*Context, io.Reader))        {}
func (s *Server) OnError(func(*Context, error))              {}
func (s *Server) OnClosed(func(*Context))                    {}

// With adds an middleware to process packets.
// Be careful when reading content from ctx.Reader(). Other middlewares and handler can't read from it again.
func (s *Server) With(func(*Context)) {}

func (s *Server) ServeHTTP(http.ResponseWriter, *http.Request) {}

// Session methods could be called in any goroutine.
type Session interface {
	ID() string
	Transport() string

	Close() error
	Store(key string, value interface{})
	Get(key string) interface{}

	// SendFrame should be called after closing last frame.
	SendFrame(FrameType) (io.WriteCloser, error)
}

type Context struct {
	context.Context
	Session    Session
	Request    *http.Request
	PacketType PacketType
	Reader     io.Reader
}

func (c *Context) Next() {}

func HTTPError(code int, msg string) error { return nil }
