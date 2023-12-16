# Sentry client for zap logger

## Integration using Sentry Client

Integration of sentry client into zap.Logger is pretty simple:
```golang
func modifyToSentryLogger(log *zap.Logger, client *sentry.Client) *zap.Logger {
	cfg := zapsentry.Configuration{
		Level: zapcore.ErrorLevel, //when to send message to sentry
		EnableBreadcrumbs: true, // enable sending breadcrumbs to Sentry 
		BreadcrumbLevel: zapcore.InfoLevel, // at what level should we sent breadcrumbs to sentry, this level can't be higher than `Level`
		Tags: map[string]string{
			"component": "system",
		},
	}
	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(client))
	
	// don't use value if error was returned. Noop core will be replaced to nil soon.
	if err != nil {
		panic(err)
	}
	
	log = zapsentry.AttachCoreToLogger(core, log)

	// if you have web service, create a new scope somewhere in middleware to have valid breadcrumbs.
	return log.With(zapsentry.NewScope())
}
```

Please note that wrapper does not guarantee that all your events will be sent before the app exits.
Flush called internally only in case of writing message with severity level > zapcore.ErrorLevel (i.e. Fatal, Panic, ...).
If you want to ensure your messages come to sentry - call the flush on native sentry client at defer. 
Example:
```golang
func main() {
	sentryClient, err := sentry.NewClient(sentry.ClientOptions{
		Dsn: "Sentry DSN",
	})
	if err != nil {
		// Handle the error here
	}
	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentryClient.Flush(2 * time.Second)
	
	// create zap log and wrapper...
	
	// create and run your app here...
}
```
