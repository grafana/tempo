package storage

import "github.com/grafana/tempo/modules/overrides"

type Overrides interface {
	MetricsGeneratorRemoteWriteHeaders(userID string) map[string]string
	MetricsGeneratorGenerateNativeHistograms(userID string) overrides.HistogramMethod
}

var _ Overrides = (overrides.Interface)(nil)
