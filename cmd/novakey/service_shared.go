package main

import "github.com/sirupsen/logrus"

func init() {
	// Basic logrus setup; on Windows, eventlog_windows.go will attach
	// an additional hook to forward logs to the Windows Event Log.
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

func LogInfo(msg string) {
	logrus.Info(msg)
}

func LogError(msg string, err error) {
	if err != nil {
		logrus.WithError(err).Error(msg)
	} else {
		logrus.Error(msg)
	}
}

// zeroBytes overwrites a slice for security best-effort.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
