package zapsentry

import "go.uber.org/zap/zapcore"

type LevelEnabler struct {
	zapcore.LevelEnabler
	enableBreadcrumbs bool
	breadcrumbsLevel  zapcore.LevelEnabler
}

func (l *LevelEnabler) Enabled(lvl zapcore.Level) bool {
	return l.LevelEnabler.Enabled(lvl) || (l.enableBreadcrumbs && l.breadcrumbsLevel.Enabled(lvl))
}
