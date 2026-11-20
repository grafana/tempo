package sharedconfig

import (
	"fmt"
	"regexp"
	"unicode/utf8"
)

const (
	MaxServiceGraphPresentationRules = 128
	MaxServiceGraphPresentationBytes = 64 * 1024
)

// ValidateServiceGraphPresentation applies the same contract to stored defaults
// and request overrides, so accepted configuration remains queryable.
func ValidateServiceGraphPresentation(services, patterns []string) error {
	if len(services)+len(patterns) > MaxServiceGraphPresentationRules {
		return fmt.Errorf("presentation rules exceed limit of %d", MaxServiceGraphPresentationRules)
	}
	size := 0
	for _, values := range [][]string{services, patterns} {
		for _, value := range values {
			if value == "" || !utf8.ValidString(value) {
				return fmt.Errorf("presentation rules must contain nonempty UTF-8 strings")
			}
			size += len(value)
			if size > MaxServiceGraphPresentationBytes {
				return fmt.Errorf("presentation rules exceed limit of %d bytes", MaxServiceGraphPresentationBytes)
			}
		}
	}
	for i, pattern := range patterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("bypass_service_regex[%d]: %w", i, err)
		}
	}
	return nil
}
