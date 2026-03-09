package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps a zap.SugaredLogger to provide structured, colored logging.
type Logger struct {
	*zap.SugaredLogger
	component string
}

// New creates a root Logger with colored console output.
func New() *Logger {
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "component",
		MessageKey:    "msg",
		EncodeTime:    zapcore.TimeEncoderOfLayout("15:04:05"),
		EncodeLevel:   zapcore.CapitalColorLevelEncoder,
		EncodeName:    zapcore.FullNameEncoder,
		ConsoleSeparator: "\t",
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.AddSync(os.Stderr),
		zapcore.DebugLevel,
	)

	return &Logger{
		SugaredLogger: zap.New(core).Sugar(),
	}
}

// Named returns a child Logger tagged with the given component name.
func (l *Logger) Named(component string) *Logger {
	return &Logger{
		SugaredLogger: l.SugaredLogger.Named(component),
		component:     component,
	}
}
