package wal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZeroFlushedTime(t *testing.T) {
	c := &CompleteBlock{}

	assert.True(t, c.FlushedTime().IsZero())
}
