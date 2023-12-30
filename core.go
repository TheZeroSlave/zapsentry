package zapsentry

import (
	"errors"
	"reflect"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultMaxBreadcrumbs = 100
	maxErrorDepth         = 10

	zapSentryScopeKey = "_zapsentry_scope_"
)

var (
	ErrInvalidBreadcrumbLevel = errors.New("breadcrumb level must be lower than or equal to error level")
)

type ClientGetter interface {
	GetClient() *sentry.Client
}

func NewScopeFromScope(scope *sentry.Scope) zapcore.Field {
	f := zap.Skip()
	f.Interface = scope
	f.Key = zapSentryScopeKey

	return f
}

func NewScope() zapcore.Field {
	return NewScopeFromScope(sentry.NewScope())
}

func NewCore(cfg Configuration, factory SentryClientFactory) (zapcore.Core, error) {
	client, err := factory()
	if err != nil {
		return zapcore.NewNopCore(), err
	}

	if cfg.EnableBreadcrumbs && zapcore.LevelOf(cfg.BreadcrumbLevel) > zapcore.LevelOf(cfg.Level) {
		return zapcore.NewNopCore(), ErrInvalidBreadcrumbLevel
	}

	if cfg.MaxBreadcrumbs <= 0 {
		cfg.MaxBreadcrumbs = defaultMaxBreadcrumbs
	}

	// copy default values to prevent accidental modification.
	matchers := make(FrameMatchers, len(defaultFrameMatchers), len(defaultFrameMatchers)+1)
	copy(matchers, defaultFrameMatchers)

	if cfg.FrameMatcher != nil {
		cfg.FrameMatcher = append(matchers, cfg.FrameMatcher)
	} else {
		cfg.FrameMatcher = matchers
	}

	var flushTimeout = time.Second * 5
	if cfg.FlushTimeout > 0 {
		flushTimeout = cfg.FlushTimeout
	}

	core := core{
		client: client,
		cfg:    &cfg,
		LevelEnabler: &LevelEnabler{
			LevelEnabler:      cfg.Level,
			breadcrumbsLevel:  cfg.BreadcrumbLevel,
			enableBreadcrumbs: cfg.EnableBreadcrumbs,
		},
		flushTimeout: flushTimeout,
		fields:       make(map[string]interface{}),
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
	clone := c.with(c.addSpecialFields(ent, fs))

	if c.cfg.EnableBreadcrumbs && c.cfg.BreadcrumbLevel.Enabled(ent.Level) {
		breadcrumb := sentry.Breadcrumb{
			Message:   ent.Message,
			Data:      clone.fields,
			Level:     sentrySeverity(ent.Level),
			Timestamp: ent.Time,
		}

		c.scope().AddBreadcrumb(&breadcrumb, c.cfg.MaxBreadcrumbs)
	}

	if c.cfg.Level.Enabled(ent.Level) {
		tagsCount := len(c.cfg.Tags)
		for _, f := range fs {
			if f.Type == zapcore.SkipType {
				if _, ok := f.Interface.(tagField); ok {
					tagsCount++
				}
			}
		}

		var hint *sentry.EventHint

		event := sentry.NewEvent()
		event.Message = ent.Message
		event.Timestamp = ent.Time
		event.Level = sentrySeverity(ent.Level)
		event.Extra = clone.fields
		event.Tags = make(map[string]string, tagsCount)
		for k, v := range c.cfg.Tags {
			event.Tags[k] = v
		}
		for _, f := range fs {
			if f.Type == zapcore.SkipType {
				switch t := f.Interface.(type) {
				case tagField:
					event.Tags[t.Key] = t.Value
				case ctxField:
					hint = &sentry.EventHint{Context: t.Value}
				}
			}
		}
		event.Exception = clone.createExceptions()

		if event.Exception == nil && !c.cfg.DisableStacktrace && c.client.Options().AttachStacktrace {
			stacktrace := sentry.NewStacktrace()
			if stacktrace != nil {
				stacktrace.Frames = c.filterFrames(stacktrace.Frames)
				event.Threads = []sentry.Thread{{Stacktrace: stacktrace, Current: true}}
			}
		}

		_ = c.client.CaptureEvent(event, hint, c.scope())
	}

	// We may be crashing the program, so should flush any buffered events.
	if ent.Level > zapcore.ErrorLevel {
		return c.Sync()
	}

	return nil
}

func (c *core) addSpecialFields(ent zapcore.Entry, fs []zapcore.Field) []zapcore.Field {
	if c.cfg.LoggerNameKey != "" && ent.LoggerName != "" {
		fs = append(fs, zap.String(c.cfg.LoggerNameKey, ent.LoggerName))
	}

	return fs
}

func (c *core) createExceptions() []sentry.Exception {
	errorsCount := len(c.errs)

	if errorsCount == 0 {
		return nil
	}

	processedErrors := make(map[string]struct{}, errorsCount)
	exceptions := make([]sentry.Exception, 0, errorsCount)

	for i := errorsCount - 1; i >= 0; i-- {
		exceptions = c.addExceptionsFromError(exceptions, processedErrors, c.errs[i])
	}

	if !c.cfg.DisableStacktrace && exceptions[0].Stacktrace == nil {
		stacktrace := sentry.NewStacktrace()
		if stacktrace != nil {
			stacktrace.Frames = c.filterFrames(stacktrace.Frames)
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

func getTypeOf(err error) string {
	return err.Error() + reflect.TypeOf(err).String()
}

func (c *core) addExceptionsFromError(
	exceptions []sentry.Exception,
	processedErrors map[string]struct{},
	err error,
) []sentry.Exception {
	for i := 0; i < maxErrorDepth && err != nil; i++ {
		if _, ok := processedErrors[getTypeOf(err)]; ok {
			return exceptions
		}

		processedErrors[getTypeOf(err)] = struct{}{}

		exception := sentry.Exception{Value: err.Error(), Type: getTypeName(err)}

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

func getTypeName(err error) string {
	switch cast := err.(type) {
	case interface{ TypeName() string }:
		return cast.TypeName()
	}
	return reflect.TypeOf(err).String()
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

// follow same logic with sentry-go to filter unnecessary frames
// ref:
// https://github.com/getsentry/sentry-go/blob/362a80dcc41f9ad11c8df556104db3efa27a419e/stacktrace.go#L256-L280
func (c *core) filterFrames(frames []sentry.Frame) []sentry.Frame {
	if len(frames) == 0 {
		return nil
	}

	for i := 0; i < len(frames); {
		if c.cfg.FrameMatcher.Matches(frames[i]) {
			if i < len(frames)-1 {
				copy(frames[i:], frames[i+1:])
			}
			frames = frames[:len(frames)-1]
			continue
		}
		i++
	}

	return frames
}
