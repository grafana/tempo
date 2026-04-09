package storage

import (
	"github.com/grafana/tempo/v2/modules/overrides"
	"github.com/grafana/tempo/v2/modules/overrides/histograms"
)

type Overrides interface {
	MetricsGeneratorRemoteWriteHeaders(userID string) map[string]string
	MetricsGeneratorGenerateNativeHistograms(userID string) histograms.HistogramMethod
}

var _ Overrides = (overrides.Interface)(nil)
