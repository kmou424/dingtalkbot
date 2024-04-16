package dingtalkbot

import (
	"github.com/charmbracelet/log"
	"os"
)

var logger = log.NewWithOptions(os.Stderr, log.Options{
	ReportTimestamp: true,
	TimeFormat:      "[2006-01-02 15:04:05.000]",
	Level:           log.InfoLevel,
	Prefix:          "dingtalkbot",
})

type iLogger struct {
}

func (l *iLogger) Debugf(format string, args ...any) {
	logger.Debugf(format, args...)
}

func (l *iLogger) Infof(format string, args ...any) {
	logger.Infof(format, args...)
}

func (l *iLogger) Warningf(format string, args ...any) {
	logger.Warnf(format, args...)
}

func (l *iLogger) Errorf(format string, args ...any) {
	logger.Errorf(format, args...)
}

func (l *iLogger) Fatalf(format string, args ...any) {
	logger.Fatalf(format, args...)
}
