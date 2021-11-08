package util

import (
	"testing"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitedLogger(t *testing.T) {
	logger := NewRateLimitedLogger(10, level.Error(log.Logger))
	assert.NotNil(t, logger)

	logger.Log("test")
}
