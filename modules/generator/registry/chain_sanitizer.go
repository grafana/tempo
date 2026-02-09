package registry

import "github.com/prometheus/prometheus/model/labels"

// ChainSanitizer applies multiple sanitizers in order.
type ChainSanitizer struct {
	sanitizers []Sanitizer
}

// NewChainSanitizer creates a Sanitizer that chains the given sanitizers in order.
// Nil sanitizers are filtered out. An empty chain is a valid noop.
func NewChainSanitizer(sanitizers ...Sanitizer) *ChainSanitizer {
	var filtered []Sanitizer
	for _, s := range sanitizers {
		if s != nil {
			filtered = append(filtered, s)
		}
	}
	return &ChainSanitizer{sanitizers: filtered}
}

func (c *ChainSanitizer) Sanitize(lbls labels.Labels) labels.Labels {
	for _, s := range c.sanitizers {
		lbls = s.Sanitize(lbls)
	}
	return lbls
}
