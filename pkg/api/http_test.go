package api

import (
	"testing"

	"github.com/grafana/tempo/cmd/tempo-query/tempo"
	"github.com/stretchr/testify/assert"
)

// For licensing reasons they strings exist in two packages. This test exists to make sure they don't
// drift.
func TestEquality(t *testing.T) {
	assert.Equal(t, AcceptHeaderKey, tempo.AcceptHeaderKey)
	assert.Equal(t, ProtobufTypeHeaderValue, tempo.ProtobufTypeHeaderValue)
}
