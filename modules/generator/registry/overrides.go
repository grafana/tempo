package registry

import (
	"time"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/overrides/histograms"
)

type Overrides interface {
	MetricsGeneratorMaxActiveSeries(userID string) uint32
	MetricsGeneratorMaxActiveEntities(userID string) uint32
	MetricsGeneratorCollectionInterval(userID string) time.Duration
	MetricsGeneratorDisableCollection(userID string) bool
	MetricsGeneratorGenerateNativeHistograms(userID string) histograms.HistogramMethod
	MetricsGeneratorTraceIDLabelName(userID string) string
	MetricsGeneratorNativeHistogramBucketFactor(userID string) float64
	MetricsGeneratorNativeHistogramMaxBucketNumber(userID string) uint32
	MetricsGeneratorNativeHistogramMinResetDuration(userID string) time.Duration
}

var _ Overrides = (overrides.Interface)(nil)
