package logger

import (
	"io"
	"io/ioutil"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/kelseyhightower/envconfig"
)

const prefix = "GO_SOCKET_IO"

var (
	global Logger
)

type Logger interface {
	Debugln(args ...interface{})
	Warnln(args ...interface{})
	Infoln(args ...interface{})
	Errorln(args ...interface{})
	Panicln(args ...interface{})
}

type config struct {
	Level    string `envconfig:"LOG_LEVEL" default:"error"`
	IsEnable bool   `envconfig:"LOG_ENABLE" default:"true"`
	IsDebug  bool   `envconfig:"DEBUG" default:"false"`
}

func init() {
	var cfg config
	envconfig.MustProcess(prefix, &cfg)

	level, err := zap.ParseAtomicLevel(cfg.Level)
	if err != nil {
		panic(err)
	}

	var opts []zap.Option
	if cfg.IsDebug {
		opts = append(opts, zap.AddStacktrace(level))
	}

	sync := ioutil.Discard

	if cfg.IsEnable {
		sync = os.Stdout
	}

	SetLogger(New(level, sync, opts...))
}

func New(level zapcore.LevelEnabler, sink io.Writer, opts ...zap.Option) *zap.SugaredLogger {
	return zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zapcore.EncoderConfig{
				TimeKey:        "ts",
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      "caller",
				MessageKey:     "message",
				StacktraceKey:  "stacktrace",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			}),
			zapcore.AddSync(sink),
			level,
		),
		opts...,
	).Sugar()
}

func SetLogger(l Logger) {
	global = l
}

func GetLogger() Logger {
	return global
}

func Debug(args ...interface{}) {
	global.Debugln(args)
}

func Warn(args ...interface{}) {
	global.Warnln(args)
}

func Info(args ...interface{}) {
	global.Infoln(args)
}

func Error(args ...interface{}) {
	global.Errorln(args)
}

func Panic(args ...interface{}) {
	global.Panicln(args)
}
