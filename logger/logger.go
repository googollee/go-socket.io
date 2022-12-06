package logger

import (
	"log"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

var l = stdr.New(log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile))

func ReplaceLogger(logger logr.Logger) {
	l = logger
}

func GetLogger(name string) logr.Logger {
	return l.WithName(name)
}
