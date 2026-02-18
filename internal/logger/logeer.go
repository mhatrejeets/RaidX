package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

func SetupAppLogging() {
	logDir := os.Getenv("APP_LOG_DIR")
	if logDir == "" {
		logDir = "./logs/app"
	}

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		logrus.SetOutput(os.Stderr)
		logrus.WithError(err).Warn("failed to create app log directory, using stderr")
		return
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("%d.log", time.Now().Unix()))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		logrus.SetOutput(os.Stderr)
		logrus.WithError(err).Warn("failed to open app log file, using stderr")
		return
	}

	logrus.SetOutput(logFile)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log.SetOutput(logFile)

	logrus.WithField("log_file", logPath).Info("app logging initialized")
}
