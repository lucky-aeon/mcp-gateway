package xlog

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

const (
	DefaultHeader = "[${level}]${prefix}[${short_file}:${line}]"
)

type Logger interface {
	echo.Logger
}

func SetHeader(name string) {
	log.SetHeader(name)
}

type logger struct {
	echo.Logger
}

func (l *logger) SetHeader(header string) {
	l.SetHeader(header)
}

func NewLogger(name string) Logger {
	lg := log.New(name)
	lg.SetLevel(log.Lvl(0))
	l := &logger{
		Logger: lg,
	}
	return l
}

func WithEchoLogger(lg echo.Logger) Logger {
	lg.SetLevel(0)
	return &logger{
		Logger: lg,
	}
}

func WithChildName(name string, lg Logger) Logger {
	l := &logger{
		Logger: lg,
	}
	l.SetPrefix(fmt.Sprintf("%s-%s", l.Prefix(), name))
	return l
}
