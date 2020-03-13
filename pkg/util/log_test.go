package util

import (
	"testing"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitedLogger(t *testing.T) {
	logger := NewRateLimitedLogger(10, level.Error(util.Logger))
	assert.NotNil(t, logger)

	logger.Log("test")
	logger.Stop()
}
