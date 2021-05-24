package log

import (
	"fmt"
	"log"
)

type Logger interface {
	Errorf(format string, v ...interface{})
	Warningf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}

func DefaultLogger() Logger {
	return &defaultLogger{
		output: log.Default(),
	}
}

type defaultLogger struct {
	output *log.Logger
}

const depth = 3

func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	l.output.Output(depth, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Warningf(format string, v ...interface{}) {
	l.output.Output(depth, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Infof(format string, v ...interface{}) {
	l.output.Output(depth, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Debugf(format string, v ...interface{}) {
	l.output.Output(depth, fmt.Sprintf(format, v...))
}
