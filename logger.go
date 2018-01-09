package zapsentry

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func AttachCoreToLogger(sentryCore zapcore.Core, l *zap.Logger) *zap.Logger {
	return l.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, sentryCore)
	}))
}
