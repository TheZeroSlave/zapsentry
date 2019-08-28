package zapsentry

import (
	"github.com/getsentry/sentry-go"
)

func NewSentryClientFromDSN(DSN string) SentryClientFactory {
	return func() (*sentry.Client, error) {
		return sentry.NewClient(sentry.ClientOptions{
			Dsn: DSN,
		})
	}
}

func NewSentryClientFromClient(client *sentry.Client) SentryClientFactory {
	return func() (*sentry.Client, error) {
		return client, nil
	}
}

type SentryClientFactory func() (*sentry.Client, error)
