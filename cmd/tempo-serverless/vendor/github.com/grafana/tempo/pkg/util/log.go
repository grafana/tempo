package util

import (
	"time"

	"github.com/go-kit/log"
	"golang.org/x/time/rate"
)

type RateLimitedLogger struct {
	limiter *rate.Limiter
	logger  log.Logger
}

func NewRateLimitedLogger(logsPerSecond int, logger log.Logger) *RateLimitedLogger {
	return &RateLimitedLogger{
		limiter: rate.NewLimiter(rate.Limit(logsPerSecond), 1),
		logger:  logger,
	}
}

func (l *RateLimitedLogger) Log(keyvals ...interface{}) {
	if !l.limiter.AllowN(time.Now(), 1) {
		return
	}

	_ = l.logger.Log(keyvals...)
}
