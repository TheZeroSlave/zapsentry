package zapsentry_test

import (
	"errors"
	"testing"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/TheZeroSlave/zapsentry"
)

func TestLevelEnabler(t *testing.T) {
	lvl := zap.NewAtomicLevelAt(zap.PanicLevel)
	core, recordedLogs := observer.New(lvl)
	logger := zap.New(core)

	var recordedSentryEvent *sentry.Event
	sentryClient := mockSentryClient(func(event *sentry.Event) {
		recordedSentryEvent = event
	})

	core, err := zapsentry.NewCore(
		zapsentry.Configuration{Level: lvl},
		zapsentry.NewSentryClientFromClient(sentryClient),
	)
	if err != nil {
		t.Fatal(err)
	}
	newLogger := zapsentry.AttachCoreToLogger(core, logger)

	newLogger.Error("foo")
	if recordedLogs.Len() > 0 || recordedSentryEvent != nil {
		t.Errorf("expected no logs before level change")
		t.Logf("logs=%v", recordedLogs.All())
		t.Logf("events=%v", recordedSentryEvent)
	}

	lvl.SetLevel(zap.ErrorLevel)
	newLogger.Error("bar")
	if recordedLogs.Len() != 1 || recordedSentryEvent == nil {
		t.Errorf("expected exactly one log after level change")
		t.Logf("logs=%v", recordedLogs.All())
		t.Logf("events=%v", recordedSentryEvent)
	}
}

func TestBreadcrumbLevelEnabler(t *testing.T) {
	corelvl := zap.NewAtomicLevelAt(zap.ErrorLevel)
	breadlvl := zap.NewAtomicLevelAt(zap.PanicLevel)

	_, err := zapsentry.NewCore(
		zapsentry.Configuration{Level: corelvl, BreadcrumbLevel: breadlvl, EnableBreadcrumbs: true},
		zapsentry.NewSentryClientFromClient(mockSentryClient(func(event *sentry.Event) {})),
	)
	if !errors.Is(err, zapsentry.ErrInvalidBreadcrumbLevel) {
		t.Errorf("expected ErrInvalidBreadcrumbLevel, got %v", err)
	}
}
