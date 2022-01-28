package log

import (
	"testing"

	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitedLogger(t *testing.T) {
	logger := NewRateLimitedLogger(10, level.Error(Logger))
	assert.NotNil(t, logger)

	logger.Log("test")
}
