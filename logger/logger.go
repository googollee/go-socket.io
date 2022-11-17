package logger

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

var l = stdr.New(nil)

func ReplaceLogger(logger logr.Logger) {
	l = logger
}

func GetLogger(name string) logr.Logger {
	return l.WithName(name)
}
