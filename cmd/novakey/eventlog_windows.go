//go:build windows
// +build windows

package main

import (
	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
)

type eventLogHook struct {
	logger service.Logger
}

func NewEventLogHook(name string) (logrus.Hook, error) {
	cfg := &service.Config{
		Name:        name,
		DisplayName: name,
		Description: name,
	}

	svc, err := service.New(nil, cfg)
	if err != nil {
		return nil, err
	}

	logger, err := svc.Logger(nil)
	if err != nil {
		return nil, err
	}

	return &eventLogHook{logger: logger}, nil
}

func (h *eventLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *eventLogHook) Fire(e *logrus.Entry) error {
	line, _ := e.String()

	switch e.Level {
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return h.logger.Error(line)
	case logrus.WarnLevel:
		return h.logger.Warning(line)
	default:
		return h.logger.Info(line)
	}
}

// init attaches the Windows Event Log hook to the global logrus logger.
func init() {
	hook, err := NewEventLogHook("NovaKey")
	if err != nil {
		// If we can't initialize the event log hook, just fall back to
		// standard logrus stderr logging.
		return
	}
	logrus.AddHook(hook)
}
