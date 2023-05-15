package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConditionIsTraceMetadata(t *testing.T) {
	require.True(t, SearchMetaCondition.IsTraceMetadata())
}
