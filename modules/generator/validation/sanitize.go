package validation

import (
	"fmt"
	"unicode/utf8"

	"github.com/prometheus/prometheus/util/strutil"
)

type SanitizeFn func(string) string

func SanitizeLabelNameWithCollisions(name string, dimensions map[string]struct{}, sanitizeFn SanitizeFn) string {
	sanitized := sanitizeFn(name)

	// check if same label as intrinsics
	if _, ok := dimensions[sanitized]; ok {
		return "__" + sanitized
	}

	return sanitized
}

func ValidateUTF8LabelValues(v []string) error {
	for _, value := range v {
		if !utf8.ValidString(value) {
			return fmt.Errorf("invalid utf8 string: %s", value)
		}
	}
	return nil
}

func SanitizeLabelName(name string) string {
	return strutil.SanitizeLabelName(name)
}
