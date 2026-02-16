package tempopb_test

import (
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

func TestQueryRangeRequest_InstantFlag(t *testing.T) {
	var q tempopb.QueryRangeRequest
	assert.False(t, q.HasInstant())

	q.SetInstant(true)
	assert.True(t, q.HasInstant())
	assert.True(t, q.GetInstant())

	q.SetInstant(false)
	assert.True(t, q.HasInstant())
	assert.False(t, q.GetInstant())
}

func TestQueryRangeRequest_MarshalUnmarshal(t *testing.T) {
	var q tempopb.QueryRangeRequest
	assert.False(t, q.HasInstant())

	q.SetInstant(true)
	q = marshalUnmarshal(t, q)
	assert.True(t, q.HasInstant())
	assert.True(t, q.GetInstant())

	q.SetInstant(false)
	q = marshalUnmarshal(t, q)
	assert.True(t, q.HasInstant())
	assert.False(t, q.GetInstant())
}

func marshalUnmarshal(t *testing.T, q tempopb.QueryRangeRequest) tempopb.QueryRangeRequest {
	marshaled, err := q.Marshal()
	assert.NoError(t, err)

	var unmarshaled tempopb.QueryRangeRequest
	err = unmarshaled.Unmarshal(marshaled)
	assert.NoError(t, err)

	return unmarshaled
}
