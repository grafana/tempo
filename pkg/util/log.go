package util

import (
	"github.com/go-kit/kit/log"
	"go.uber.org/ratelimit"
)

type RateLimitedLogger struct {
	logChan chan []interface{}
}

func NewRateLimitedLogger(logsPerSecond int, logger log.Logger) *RateLimitedLogger {
	r := &RateLimitedLogger{
		logChan: make(chan []interface{}),
	}

	go func() {
		limiter := ratelimit.New(logsPerSecond)
		for keyvals := range r.logChan {
			_ = logger.Log(keyvals...)
			limiter.Take()
		}
	}()

	return r
}

func (l *RateLimitedLogger) Log(keyvals ...interface{}) {
	select {
	case l.logChan <- keyvals:
	default:
	}
}

func (l *RateLimitedLogger) Stop() {
	close(l.logChan)
}
