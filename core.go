package zapsentry

import (
	raven "github.com/getsentry/raven-go"
	"go.uber.org/zap/zapcore"
)

const (
	traceContextLines = 3
	traceSkipFrames   = 2
)

func NewCore(cfg Configuration, factory SentryClientFactory) (zapcore.Core, error) {
	client, err := factory()
	if err != nil {
		return zapcore.NewNopCore(), err
	}
	return &core{
		client:       client,
		cfg:          &cfg,
		LevelEnabler: cfg.Level,
		fields:       make(map[string]interface{}),
	}, nil
}

func (c *core) With(fs []zapcore.Field) zapcore.Core {
	return c.with(fs)
}

func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.cfg.Level.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *core) Write(ent zapcore.Entry, fs []zapcore.Field) error {
	clone := c.with(fs)

	packet := &raven.Packet{
		Message:   ent.Message,
		Timestamp: raven.Timestamp(ent.Time),
		Level:     ravenSeverity(ent.Level),
		Platform:  "Golang",
		Extra:     clone.fields,
	}

	if !c.cfg.DisableStacktrace {
		trace := raven.NewStacktrace(traceSkipFrames, traceContextLines, nil)
		if trace != nil {
			packet.Interfaces = append(packet.Interfaces, trace)
		}
	}

	_, _ = c.client.Capture(packet, c.cfg.Tags)

	// We may be crashing the program, so should flush any buffered events.
	if ent.Level > zapcore.ErrorLevel {
		c.client.Wait()
	}
	return nil
}

func (c *core) Sync() error {
	c.client.Wait()
	return nil
}

func (c *core) with(fs []zapcore.Field) *core {
	// Copy our map.
	m := make(map[string]interface{}, len(c.fields))
	for k, v := range c.fields {
		m[k] = v
	}

	// Add fields to an in-memory encoder.
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fs {
		f.AddTo(enc)
	}

	// Merge the two maps.
	for k, v := range enc.Fields {
		m[k] = v
	}

	return &core{
		client:       c.client,
		cfg:          c.cfg,
		fields:       m,
		LevelEnabler: c.LevelEnabler,
	}
}

type ClientGetter interface {
	GetClient() *raven.Client
}

func (c *core) GetClient() *raven.Client {
	return c.client
}

type core struct {
	client *raven.Client
	cfg    *Configuration
	zapcore.LevelEnabler

	fields map[string]interface{}
}
