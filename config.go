package zapsentry

import (
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

// Configuration is a minimal set of parameters for Sentry integration.
type Configuration struct {
	// Tags are passed as is to the corresponding sentry.Event field.
	Tags map[string]string

	// LoggerNameKey is the key for zap logger name.
	// If not empty, the name is added to the rest of zapcore.Field(s),
	// so that be careful with key duplicates.
	// Leave LoggerNameKey empty to disable the feature.
	LoggerNameKey string

	// DisableStacktrace disables adding stacktrace to sentry.Event, if set.
	DisableStacktrace bool

	// Level is the minimal level of sentry.Event(s).
	Level zapcore.LevelEnabler

	// EnableBreadcrumbs enables use of sentry.Breadcrumb(s).
	// This feature works only when you explicitly passed new scope.
	EnableBreadcrumbs bool

	// If you cannot (or don't want to) create a local scope for breadcrumbs
	// you an disable the default behavior of ignoring the global scope
	EnableBreadcrumbsInGlobalScope bool

	// BreadcrumbLevel is the minimal level of sentry.Breadcrumb(s).
	// Breadcrumb specifies an application event that occurred before a Sentry event.
	// NewCore fails if BreadcrumbLevel is greater than Level.
	// The field is ignored, if EnableBreadcrumbs is not set.
	BreadcrumbLevel zapcore.LevelEnabler

	// MaxBreadcrumbs is the maximum number of breadcrumb events to keep.
	// Leave it zero or set to negative for a reasonable default value.
	// The field is ignored, if EnableBreadcrumbs is not set.
	MaxBreadcrumbs int

	// FlushTimeout is the timeout for flushing events to Sentry.
	FlushTimeout time.Duration

	// Hub overrides the sentry.CurrentHub value.
	// See sentry.Hub docs for more detail.
	Hub *sentry.Hub

	// FrameMatcher allows to ignore some frames of the stack trace.
	// this is particularly useful when you want to ignore for instances frames from convenience wrappers
	FrameMatcher FrameMatcher
}
