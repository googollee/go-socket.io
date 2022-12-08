package logger

import (
	"os"
	"strconv"
)

const (
	envLogLevel  = "LOG_LEVEL"
	envLogEnable = "LOG_ENABLE"
	envDebug     = "DEBUG"
)

func envString(env string, def string) string {
	if s := os.Getenv(env); s != "" {
		return s
	}

	return def
}

func envBool(env string, def bool) bool {
	b, err := strconv.ParseBool(os.Getenv(env))
	if err != nil {
		return def
	}

	return b
}
