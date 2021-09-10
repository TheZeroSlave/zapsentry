package zapsentry_test

import (
	"fmt"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/TheZeroSlave/zapsentry"
)

func ExampleAttachCoreToLogger() {
	// Setup zap with observer (for testing), originally we use
	// 		config = zap.NewDevelopmentConfig()
	//  	logger, err := config.Build()
	// to build zap logger, here we use zap/zaptest/observer for testing
	core, recordedLogs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core, zap.AddStacktrace(zap.DebugLevel))

	// Setup mock sentry client for testing, in general we use sentry.NewClient
	var recordedSentryEvent *sentry.Event
	sentryClient := mockSentryClient(func(event *sentry.Event) {
		recordedSentryEvent = event
	})

	// Setup zapsentry
	core, err := zapsentry.NewCore(zapsentry.Configuration{
		Level: zapcore.ErrorLevel, // when to send message to sentry
		EnableBreadcrumbs: true, // enable sending breadcrumbs to Sentry
		BreadcrumbLevel: zapcore.InfoLevel, // at what level should we sent breadcrumbs to sentry
		Tags: map[string]string{
			"component": "system",
		},
	}, zapsentry.NewSentryClientFromClient(sentryClient))
	if err != nil {
		log.Fatal(err)
	}
	newLogger := zapsentry.AttachCoreToLogger(core, logger)

	// Send error log
	newLogger.
		With(zapsentry.NewScope()).
		Error("[error] something went wrong!", zap.String("method", "unknown"))

	// Check output
	fmt.Println(recordedLogs.All()[0].Message)
	fmt.Println(recordedSentryEvent.Message)
	fmt.Println(recordedSentryEvent.Extra)
	// Output: [error] something went wrong!
	// [error] something went wrong!
	// map[method:unknown]
}

func mockSentryClient(f func(event *sentry.Event)) *sentry.Client {
	client, _ := sentry.NewClient(sentry.ClientOptions{
		Dsn:              "",
		Transport:        &transport{MockSendEvent: f},
	})
	return client
}

type transport struct {
	MockSendEvent func(event *sentry.Event)
}

// Flush waits until any buffered events are sent to the Sentry server, blocking
// for at most the given timeout. It returns false if the timeout was reached.
func (f *transport) Flush(_ time.Duration) bool { return true }

// Configure is called by the Client itself, providing it it's own ClientOptions.
func (f *transport) Configure(_ sentry.ClientOptions) {}

// SendEvent assembles a new packet out of Event and sends it to remote server.
// We use this method to capture the event for testing
func (f *transport) SendEvent(event *sentry.Event) {
	f.MockSendEvent(event)
}

