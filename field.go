package zapsentry

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type tagField string

// Tag adds a Sentry tag to the event. Tags attached via [zap.Logger.With] are
// carried over to every event logged through the derived logger.
func Tag(key string, value string) zap.Field {
	return zap.Field{Key: key, Type: zapcore.SkipType, Interface: tagField(value)}
}

type ctxField struct {
	Value context.Context
}

// Context adds a context to the logger.
// This can be used e.g. to pass trace information to sentry and allow linking events to their respective traces.
//
// When attached via [zap.Logger.With], the context is retained for the lifetime of the
// derived logger, so prefer passing request-scoped contexts at the log call site.
//
// See also https://docs.sentry.io/platforms/go/performance/instrumentation/opentelemetry/#linking-errors-to-transactions
func Context(ctx context.Context) zap.Field {
	return zap.Field{Key: "context", Type: zapcore.SkipType, Interface: ctxField{ctx}}
}
