// Package fftypes contains core types used throughout the functions framework
// (ff) implementation.
package fftypes

import (
	"cloud.google.com/go/functions/metadata"
)

// BackgroundEvent is the incoming payload to functions framework to trigger
// Background Functions (https://cloud.google.com/functions/docs/writing/background).
// These fields are converted into parameters passed to the user function.
type BackgroundEvent struct {
	Data     interface{}        `json:"data"`
	Metadata *metadata.Metadata `json:"context"`
}
