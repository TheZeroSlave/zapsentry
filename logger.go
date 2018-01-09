package zapsentry

import (
"go.uber.org/zap/zapcore"
"go.uber.org/zap"
)

func AttachCoreToLogger(sentryCore zapcore.Core, l *zap.Logger) *zap.Logger {
	return l.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, sentryCore)
	}))
}