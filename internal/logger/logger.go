package logger

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	logrus.FieldLogger
	SetLevel(level logrus.Level)
	GetLevel() logrus.Level
}

func New(ctx context.Context, level logrus.Level, output *os.File) Logger {
	log := logrus.New()
	log.SetLevel(level)
	log.SetOutput(output)

	log.WithContext(ctx)

	return log
}
