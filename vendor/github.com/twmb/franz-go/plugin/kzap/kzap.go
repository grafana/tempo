// Package kzap provides a plug-in kgo.Logger wrapping uber's zap for usage in
// a kgo.Client.
//
// This can be used like so:
//
//	cl, err := kgo.NewClient(
//	        kgo.WithLogger(kzap.New(zapLogger)),
//	        // ...other opts
//	)
//
// By default, the logger chooses the highest level possible that is enabled on
// the zap logger, and then sticks with that level forever. A variable level
// can be chosen by specifying the LevelFn option. See the documentation on
// Level or LevelFn for more info.
package kzap

import (
	"go.uber.org/zap"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Logger provides the kgo.Logger interface for usage in kgo.WithLogger when
// initializing a client.
type Logger struct {
	zl *zap.Logger

	levelFn func() kgo.LogLevel
}

// New returns a new logger that checks the enabled log level on every log.
func New(zl *zap.Logger, opts ...Opt) *Logger {
	c := zl.Core()
	l := &Logger{
		zl: zl,
		levelFn: func() kgo.LogLevel {
			switch {
			case c.Enabled(zap.DebugLevel):
				return kgo.LogLevelDebug
			case c.Enabled(zap.InfoLevel):
				return kgo.LogLevelInfo
			case c.Enabled(zap.WarnLevel):
				return kgo.LogLevelWarn
			case c.Enabled(zap.ErrorLevel):
				return kgo.LogLevelError
			}
			return kgo.LogLevelNone // default
		},
	}
	for _, opt := range opts {
		opt.apply(l)
	}
	return l
}

// Opt applies options to the logger.
type Opt interface {
	apply(*Logger)
}

type opt struct{ fn func(*Logger) }

func (o opt) apply(l *Logger) { o.fn(l) }

// LevelFn sets a function that can dynamically change the log level. You may
// want to set this is the checking if a log level is enabled is expensive.
func LevelFn(fn func() kgo.LogLevel) Opt {
	return opt{func(l *Logger) { l.levelFn = fn }}
}

// AtomicLevel returns an option that uses the current atomic level for
// LevelFn. If your zap logger uses the AtomicLevel already, using this option
// is not necessary, but it is *slightly* less work than the default level
// function that has to check if each level is enabled individually.
func AtomicLevel(level zap.AtomicLevel) Opt {
	return LevelFn(func() kgo.LogLevel {
		switch level.Level() {
		case zap.DebugLevel:
			return kgo.LogLevelDebug
		case zap.InfoLevel:
			return kgo.LogLevelInfo
		case zap.WarnLevel:
			return kgo.LogLevelWarn
		case zap.ErrorLevel:
			return kgo.LogLevelError
		}
		return kgo.LogLevelNone
	})
}

// Level sets a static level for the kgo.Logger Level function.
func Level(level kgo.LogLevel) Opt {
	return LevelFn(func() kgo.LogLevel { return level })
}

// Level is for the kgo.Logger interface.
func (l *Logger) Level() kgo.LogLevel {
	return l.levelFn()
}

// Log is for the kgo.Logger interface.
func (l *Logger) Log(level kgo.LogLevel, msg string, keyvals ...any) {
	fields := make([]zap.Field, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		k, v := keyvals[i], keyvals[i+1]
		fields = append(fields, zap.Any(k.(string), v))
	}
	switch level {
	case kgo.LogLevelDebug:
		l.zl.Debug(msg, fields...)
	case kgo.LogLevelError:
		l.zl.Error(msg, fields...)
	case kgo.LogLevelInfo:
		l.zl.Info(msg, fields...)
	case kgo.LogLevelWarn:
		l.zl.Warn(msg, fields...)
	default:
		// do nothing
	}
}
