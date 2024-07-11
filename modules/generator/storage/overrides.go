package storage

type Overrides interface {
	MetricsGeneratorRemoteWriteHeaders(userID string) map[string]string
	MetricsGeneratorGenerateNativeHistograms(userID string) string
}
