package logger

import (
	"golang.org/x/exp/slog"
)

var Log *slog.Logger = slog.Default()

func Error(msg string, err error) {
	Log.Error(msg, "err", err.Error())
}

func Info(msg string, args ...interface{}) {
	Log.Info(msg, args...)
}
