package zapsentry

import (
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	maxBreadcrumbs = 1000
	maxErrorDepth  = 10

	zapSentryScopeKey = "_zapsentry_scope_"
)

func NewScope() zapcore.Field {
	f := zap.Skip()
	f.Interface = sentry.NewScope()
	f.Key = zapSentryScopeKey

	return f
}

func NewCore(cfg Configuration, factory SentryClientFactory) (zapcore.Core, error) {
	client, err := factory()
	if err != nil {
		return zapcore.NewNopCore(), err
	}

	if cfg.EnableBreadcrumbs && cfg.BreadcrumbLevel > cfg.Level {
		return zapcore.NewNopCore(), errors.New("breadcrumb level must be lower than error level")
	}

	core := core{
		client: client,
		cfg:    &cfg,
		LevelEnabler: &LevelEnabler{
			Level:             cfg.Level,
			breadcrumbsLevel:  cfg.BreadcrumbLevel,
			enableBreadcrumbs: cfg.EnableBreadcrumbs,
		},
		flushTimeout: 5 * time.Second,
		fields:       make(map[string]interface{}),
	}

	if cfg.FlushTimeout > 0 {
		core.flushTimeout = cfg.FlushTimeout
	}

	return &core, nil
}

func (c *core) With(fs []zapcore.Field) zapcore.Core {
	return c.with(fs)
}

func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.cfg.EnableBreadcrumbs && c.cfg.BreadcrumbLevel.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	if c.cfg.Level.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *core) Write(ent zapcore.Entry, fs []zapcore.Field) error {
	clone := c.with(fs)

	// only when we have local sentryScope to avoid collecting all breadcrumbs ever in a global scope
	if c.cfg.EnableBreadcrumbs && c.cfg.BreadcrumbLevel.Enabled(ent.Level) && c.sentryScope != nil {
		breadcrumb := sentry.Breadcrumb{
			Message:   ent.Message,
			Data:      clone.fields,
			Level:     sentrySeverity(ent.Level),
			Timestamp: ent.Time,
		}

		c.sentryScope.AddBreadcrumb(&breadcrumb, maxBreadcrumbs)
	}

	if c.cfg.Level.Enabled(ent.Level) {
		event := sentry.NewEvent()
		event.Message = ent.Message
		event.Timestamp = ent.Time
		event.Level = sentrySeverity(ent.Level)
		event.Extra = clone.fields
		event.Tags = c.cfg.Tags
		event.Exception = clone.createExceptions()

		if event.Exception == nil && !c.cfg.DisableStacktrace && c.client.Options().AttachStacktrace {
			stacktrace := sentry.NewStacktrace()
			if stacktrace != nil {
				stacktrace.Frames = filterFrames(stacktrace.Frames)
				event.Threads = []sentry.Thread{{Stacktrace: stacktrace, Current: true}}
			}
		}

		_ = c.client.CaptureEvent(event, nil, c.scope())
	}

	// We may be crashing the program, so should flush any buffered events.
	if ent.Level > zapcore.ErrorLevel {
		return c.Sync()
	}

	return nil
}

func (c *core) createExceptions() []sentry.Exception {
	errorsCount := len(c.errs)

	if errorsCount == 0 {
		return nil
	}

	processedErrors := make(map[error]struct{}, errorsCount)
	exceptions := make([]sentry.Exception, 0, errorsCount)

	for i := errorsCount - 1; i >= 0; i-- {
		exceptions = c.addExceptionsFromError(exceptions, processedErrors, c.errs[i])
	}

	if !c.cfg.DisableStacktrace && exceptions[0].Stacktrace == nil {
		stacktrace := sentry.NewStacktrace()
		if stacktrace != nil {
			stacktrace.Frames = filterFrames(stacktrace.Frames)
			exceptions[0].Stacktrace = stacktrace
		}
	}

	// Reverse the exceptions; the most recent error must be the last one
	for i := len(exceptions)/2 - 1; i >= 0; i-- {
		j := len(exceptions) - 1 - i
		exceptions[i], exceptions[j] = exceptions[j], exceptions[i]
	}

	return exceptions
}

func (c *core) addExceptionsFromError(
	exceptions []sentry.Exception,
	processedErrors map[error]struct{},
	err error,
) []sentry.Exception {
	for i := 0; i < maxErrorDepth && err != nil; i++ {
		if _, ok := processedErrors[err]; ok {
			return exceptions
		}

		processedErrors[err] = struct{}{}

		exception := sentry.Exception{Value: err.Error(), Type: reflect.TypeOf(err).String()}

		if !c.cfg.DisableStacktrace {
			exception.Stacktrace = sentry.ExtractStacktrace(err)
		}

		exceptions = append(exceptions, exception)

		switch previousProvider := err.(type) {
		case interface{ Unwrap() error }:
			err = previousProvider.Unwrap()
		case interface{ Cause() error }:
			err = previousProvider.Cause()
		default:
			err = nil
		}
	}

	return exceptions
}

func (c *core) hub() *sentry.Hub {
	if c.cfg.Hub != nil {
		return c.cfg.Hub
	}

	return sentry.CurrentHub()
}

func (c *core) scope() *sentry.Scope {
	if c.sentryScope != nil {
		return c.sentryScope
	}

	return c.hub().Scope()
}

func getScope(field zapcore.Field) *sentry.Scope {
	if field.Type == zapcore.SkipType {
		if scope, ok := field.Interface.(*sentry.Scope); ok && field.Key == zapSentryScopeKey {
			return scope
		}
	}

	return nil
}

func (c *core) Sync() error {
	c.client.Flush(c.flushTimeout)

	return nil
}

func (c *core) with(fs []zapcore.Field) *core {
	if len(fs) == 0 {
		return c
	}

	errs := make([]error, len(c.errs))

	copy(errs, c.errs)

	fields := make(map[string]interface{}, len(c.fields)+len(fs))

	for k, v := range c.fields {
		fields[k] = v
	}

	sentryScope := c.sentryScope
	enc := zapcore.NewMapObjectEncoder()

	for _, f := range fs {
		f.AddTo(enc)

		if f.Type == zapcore.ErrorType {
			errs = append(errs, f.Interface.(error))
		} else if errSlice, ok := f.Interface.([]error); ok {
			errs = append(errs, errSlice...)
		} else if scope := getScope(f); scope != nil {
			sentryScope = scope
		}
	}

	for k, v := range enc.Fields {
		fields[k] = v
	}

	return &core{
		client:       c.client,
		cfg:          c.cfg,
		LevelEnabler: c.LevelEnabler,
		flushTimeout: c.flushTimeout,
		sentryScope:  sentryScope,
		errs:         errs,
		fields:       fields,
	}
}

type ClientGetter interface {
	GetClient() *sentry.Client
}

func (c *core) GetClient() *sentry.Client {
	return c.client
}

type core struct {
	client *sentry.Client
	cfg    *Configuration
	zapcore.LevelEnabler
	flushTimeout time.Duration

	sentryScope *sentry.Scope

	errs   []error
	fields map[string]interface{}
}

type LevelEnabler struct {
	zapcore.Level
	enableBreadcrumbs bool
	breadcrumbsLevel  zapcore.Level
}

func (l *LevelEnabler) Enabled(lvl zapcore.Level) bool {
	return l.Level.Enabled(lvl) || (l.enableBreadcrumbs && l.breadcrumbsLevel.Enabled(lvl))
}

// follow same logic with sentry-go to filter unnecessary frames
// ref:
// https://github.com/getsentry/sentry-go/blob/362a80dcc41f9ad11c8df556104db3efa27a419e/stacktrace.go#L256-L280
func filterFrames(frames []sentry.Frame) []sentry.Frame {
	if len(frames) == 0 {
		return nil
	}

	for i := range frames {
		// Skip zapsentry and zap internal frames, except for frames in _test packages (for
		// testing).
		if (strings.HasPrefix(frames[i].Module, "github.com/TheZeroSlave/zapsentry") ||
			strings.HasPrefix(frames[i].Function, "go.uber.org/zap")) &&
			!strings.HasSuffix(frames[i].Module, "_test") {
			return frames[0:i]
		}
	}

	return frames
}
