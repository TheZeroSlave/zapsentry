package zapsentry_test

import (
	"context"
	"testing"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type captured struct {
	events []*sentry.Event
	hints  []*sentry.EventHint
}

func newTestLogger(t *testing.T) (*zap.Logger, *captured) {
	t.Helper()
	c := &captured{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Transport: &transport{MockSendEvent: func(e *sentry.Event) { c.events = append(c.events, e) }},
		BeforeSend: func(e *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			c.hints = append(c.hints, hint)
			return e
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	core, err := zapsentry.NewCore(
		zapsentry.Configuration{Level: zapcore.ErrorLevel, DisableStacktrace: true, Tags: map[string]string{"from_cfg": "c"}},
		zapsentry.NewSentryClientFromClient(client),
	)
	if err != nil {
		t.Fatal(err)
	}
	return zap.New(core), c
}

func TestTagViaWith(t *testing.T) {
	log, c := newTestLogger(t)

	log.With(zapsentry.Tag("from_with", "a")).Error("boom", zapsentry.Tag("from_call", "b"))

	if len(c.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(c.events))
	}
	want := map[string]string{"from_cfg": "c", "from_with": "a", "from_call": "b"}
	for k, v := range want {
		if c.events[0].Tags[k] != v {
			t.Errorf("tag %q: got %q, want %q (all: %v)", k, c.events[0].Tags[k], v, c.events[0].Tags)
		}
	}
}

func TestContextViaWith(t *testing.T) {
	log, c := newTestLogger(t)

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "marker")

	log.With(zapsentry.Context(ctx)).Error("boom")

	if len(c.hints) != 1 {
		t.Fatalf("expected 1 hint, got %d", len(c.hints))
	}
	hint := c.hints[0]
	if hint == nil || hint.Context == nil {
		t.Fatalf("With() context missing from hint: %+v", hint)
	}
	if got := hint.Context.Value(ctxKey{}); got != "marker" {
		t.Errorf("unexpected context value: %v", got)
	}
}
