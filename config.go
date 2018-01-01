package zapsentry

import (
	"go.uber.org/zap/zapcore"
	"github.com/getsentry/raven-go"
)

// Configuration is a minimal set of parameters for Sentry integration.
type Configuration struct {
	DSN   string
	Tags  map[string]string
	DisableStacktrace bool
	Level zapcore.Level
}

func ravenSeverity(lvl zapcore.Level) raven.Severity {
	switch lvl {
	case zapcore.DebugLevel:
		return raven.INFO
	case zapcore.InfoLevel:
		return raven.INFO
	case zapcore.WarnLevel:
		return raven.WARNING
	case zapcore.ErrorLevel:
		return raven.ERROR
	case zapcore.DPanicLevel:
		return raven.FATAL
	case zapcore.PanicLevel:
		return raven.FATAL
	case zapcore.FatalLevel:
		return raven.FATAL
	default:
		// Unrecognized levels are fatal.
		return raven.FATAL
	}
}
