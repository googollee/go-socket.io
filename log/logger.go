package log

import (
	"fmt"
	"log"
	"os"
)

// Logger logs messages with different levels.
type Logger interface {
	Errorf(format string, v ...interface{})
	Warningf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}

// DefaultLogger returns a default logger which outputs to stderr.
func DefaultLogger() Logger {
	return &defaultLogger{
		output: log.New(os.Stderr, "", log.LstdFlags),
	}
}

type defaultLogger struct {
	output *log.Logger
}

const depth = 3

func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	_ = l.output.Output(depth, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Warningf(format string, v ...interface{}) {
	_ = l.output.Output(depth, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Infof(format string, v ...interface{}) {
	_ = l.output.Output(depth, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Debugf(format string, v ...interface{}) {
	_ = l.output.Output(depth, fmt.Sprintf(format, v...))
}
