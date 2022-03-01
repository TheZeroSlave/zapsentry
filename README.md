# Sentry client for zap logger

## Integration using Sentry DSN

Integration of sentry client into zap.Logger is pretty simple:
```golang
func modifyToSentryLogger(log *zap.Logger, DSN string) *zap.Logger {
	cfg := zapsentry.Configuration{
		Level: zapcore.ErrorLevel, //when to send message to sentry
		EnableBreadcrumbs: true, // enable sending breadcrumbs to Sentry 
		BreadcrumbLevel: zapcore.InfoLevel, // at what level should we sent breadcrumbs to sentry
		Tags: map[string]string{
			"component": "system",
		},
	}
	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromDSN(DSN))
	
	// to use breadcrumbs feature - create new scope explicitly
	log = log.With(zapsentry.NewScope())
	
	//in case of err it will return noop core. so we can safely attach it
	if err != nil {
		log.Warn("failed to init zap", zap.Error(err))
	}
	return zapsentry.AttachCoreToLogger(core, log)
}
```

## Integraiton using Sentry Client

Integration of sentry client into zap.Logger is pretty simple:
```golang
func modifyToSentryLogger(log *zap.Logger, client *sentry.Client) *zap.Logger {
	cfg := zapsentry.Configuration{
		Level: zapcore.ErrorLevel, //when to send message to sentry
		EnableBreadcrumbs: true, // enable sending breadcrumbs to Sentry 
		BreadcrumbLevel: zapcore.InfoLevel, // at what level should we sent breadcrumbs to sentry
		Tags: map[string]string{
			"component": "system",
		},
	}
	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(client))
	
	// to use breadcrumbs feature - create new scope explicitly
	log = log.With(zapsentry.NewScope())
	
	//in case of err it will return noop core. so we can safely attach it
	if err != nil {
		log.Warn("failed to init zap", zap.Error(err))
	}
	return zapsentry.AttachCoreToLogger(core, log)
}
```

Please note that both examples does not guarantee that your events will be sent before the app exists.
To ensure this, the easy way is to use the client example and defer the flush. Example:
```golang
    sentryClient, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:         "Sentry DSN",
	})
	if err != nil {
		// Handle the error here
	}

	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentryClient.Flush(2 * time.Second)
```