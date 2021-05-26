package engineio

import (
	"io"
	"net/http"
	"time"

	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/log"
)

type Options func(*Server)

func OptionPingInterval(time.Duration) Options                                      { return nil }
func OptionPingTimeout(time.Duration) Options                                       { return nil }
func OptionMaxBufferSize(int) Options                                               { return nil }
func OptionLogger(logger log.Logger) Options                                        { return nil }
func OptionTransports(initial transport.Name, upgradings ...transport.Name) Options { return nil }
func OptionJSONP(padding int) Options                                               { return nil }

type Server struct{}

func New(...Options) (*Server, error) {
	return nil, nil
}

// OnXXX should be called before serving HTTP.
// The engineio framework processes next messages after OnXXX() done. All callback passing to OnXXX should return ASAP.
func (s *Server) OnOpen(func(*Context) error)         {}
func (s *Server) OnMessage(func(*Context, io.Reader)) {}
func (s *Server) OnError(func(*Context, error))       {}
func (s *Server) OnClose(func(*Context))              {}

// With adds an middleware to process packets.
// Be careful when reading content from ctx.Reader(). Other middlewares and handler can't read from it again.
func (s *Server) With(func(*Context, *Packet)) {}

func (s *Server) ServeHTTP(http.ResponseWriter, *http.Request) {}
