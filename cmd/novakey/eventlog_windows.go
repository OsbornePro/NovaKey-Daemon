// Copyright © 2025 Robert H. Osborne (OsbornePro)
// All rights reserved.
//
// This file is part of the NovaKey software suite.
// Unauthorized copying, distribution, modification, or reverse‑engineering
// of this source code, in whole or in part, is strictly prohibited.
//
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
	// Create a dummy service.Config so we can get a logger bound to the name.
	cfg := &service.Config{
		Name:        name,
		DisplayName: name,
		Description: name,
	}

	// Create a new service instance (not installed, just used for logging)
	svc, err := service.New(nil, cfg)
	if err != nil {
		return nil, err
	}

	// Retrieve the platform-specific logger (Event Log on Windows)
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
