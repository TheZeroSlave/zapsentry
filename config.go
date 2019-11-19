package zapsentry

import (
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

// Configuration is a minimal set of parameters for Sentry integration.
type Configuration struct {
	Tags              map[string]string
	DisableStacktrace bool
	Level             zapcore.Level
	FlushTimeout      time.Duration
	Hub               *sentry.Hub
}
