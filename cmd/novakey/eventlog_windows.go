// Copyright © 2025 Tobor (IT Senior Systems Engineer)
// All rights reserved.
//
// This file is part of the PassLink software suite.
// Unauthorized copying, distribution, modification, or reverse‑engineering
// of this source code, in whole or in part, is strictly prohibited.
//
//go:build windows
// +build windows

package main

import (
	"github.com/sirupsen/logrus"

	"golang.org/x/sys/windows/svc/eventlog"
)

type eventLogHook struct{ w *eventlog.Writer }

func NewEventLogHook(name string) (logrus.Hook, error) {
	w, err := eventlog.Open(name)
	if err != nil {
		return nil, err
	}
	return &eventLogHook{w}, nil
}

func (h *eventLogHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *eventLogHook) Fire(e *logrus.Entry) error {
	line, _ := e.String()
	switch e.Level {
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return h.w.Error(1, line)
	case logrus.WarnLevel:
		return h.w.Warning(1, line)
	default:
		return h.w.Info(1, line)
	}
}
