package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// New creates a pre-configured logrus logger with JSON formatting.
// All application logs are structured to be machine-parseable.
func New(level string) *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})

	parsed, err := logrus.ParseLevel(level)
	if err != nil {
		parsed = logrus.InfoLevel
	}
	log.SetLevel(parsed)

	return log
}
