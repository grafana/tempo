package tempopb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheableAdditionalMetrics(t *testing.T) {
	assert.Nil(t, CacheableAdditionalMetrics(nil))
	assert.Nil(t, CacheableAdditionalMetrics(map[string]int64{}))
	assert.Nil(t, CacheableAdditionalMetrics(map[string]int64{
		AdditionalMetricCacheHits: 5,
	}))

	got := CacheableAdditionalMetrics(map[string]int64{
		AdditionalMetricCacheHits:   5,
		AdditionalMetricEngineBytes: 42,
	})
	assert.Equal(t, map[string]int64{AdditionalMetricEngineBytes: 42}, got)
}
