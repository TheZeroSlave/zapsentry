package zapsentry

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type tagField struct {
	Key   string
	Value string
}

func Tag(key string, value string) zap.Field {
	return zap.Field{Key: key, Type: zapcore.SkipType, Interface: tagField{key, value}}
}
