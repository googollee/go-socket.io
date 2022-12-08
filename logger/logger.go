package logger

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

var (
	l *logger
)

type logger struct {
	logger *logrus.Logger
	opts   *options
}

// DEBUG = true debug level by default
// DEBUG = false -- default value by error message
// LOG_LEVEL = info
// LOG_ENABLE = true // false disable any server logs
// fatal level never used, because fatal method call os.Exist(1) without closing resources.

type options struct {
	isEnable bool
	level    string
}

func init() {
	log := logrus.New()

	log.SetFormatter(&logrus.JSONFormatter{})

	level, err := logrus.ParseLevel(envString(envLogLevel, "error"))
	if err != nil || level == logrus.FatalLevel {
		panic("not supported log level")
	}

	log.SetLevel(level)
	if !l.opts.isEnable {
		log.Out = ioutil.Discard
	}

	l.logger = log
}

func Debug(args ...interface{}) {
	l.logger.Debugln(args)
}

func Warn(args ...interface{}) {
	l.logger.Warnln(args)
}

func Info(args ...interface{}) {
	l.logger.Infoln(args)
}

func Error(args ...interface{}) {
	l.logger.Errorln(args)
}

func Panic(args ...interface{}) {
	l.logger.Panicln(args)
}
