package core

import (
	"fmt"
	"io"
	"log/slog"
)

type SlogLogger struct {
	logger *slog.Logger
}

func NewSlogLogger(w io.Writer) Logger {
	return &SlogLogger{
		logger: slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})),
	}
}

func (l *SlogLogger) Criticalf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...), slog.String("severity", "critical"))
}

func (l *SlogLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

func (l *SlogLogger) Errorf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

func (l *SlogLogger) Noticef(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...), slog.String("severity", "notice"))
}

func (l *SlogLogger) Warningf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}
