package zapsentry

import "github.com/getsentry/raven-go"

func NewSentryClientFromDSN(DSN string) SentryClientFactory {
	return func() (*raven.Client, error) {
		return raven.New(DSN)
	}
}

func NewSentryClientFromClient(client *raven.Client) SentryClientFactory {
	return func() (*raven.Client, error) {
		return client, nil
	}
}

type SentryClientFactory func() (*raven.Client, error)
