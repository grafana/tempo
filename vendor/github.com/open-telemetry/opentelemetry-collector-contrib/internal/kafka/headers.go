// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafka // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"

import (
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/propagation"
)

// HeaderCarrier adapts a kgo.RecordHeader slice to propagation.TextMapCarrier.
type HeaderCarrier []kgo.RecordHeader

var _ propagation.TextMapCarrier = (*HeaderCarrier)(nil)

// Get returns the value of the first header with the given key.
func (c *HeaderCarrier) Get(key string) string {
	for _, h := range *c {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set sets the value of the header with the given key, appending a new
// header if none exists.
func (c *HeaderCarrier) Set(key, value string) {
	for i, h := range *c {
		if h.Key == key {
			(*c)[i].Value = []byte(value)
			return
		}
	}
	*c = append(*c, kgo.RecordHeader{Key: key, Value: []byte(value)})
}

// Keys returns the keys of all headers, including duplicates.
func (c *HeaderCarrier) Keys() []string {
	keys := make([]string, len(*c))
	for i, h := range *c {
		keys[i] = h.Key
	}
	return keys
}
