// Package histograms provides types and constants for histogram configuration.
// This is used in the base overrides, as well as the user configurable
// overrides API.
package histograms

type HistogramMethod string

const (
	HistogramMethodClassic HistogramMethod = "classic"
	HistogramMethodNative  HistogramMethod = "native"
	HistogramMethodBoth    HistogramMethod = "both"
)
