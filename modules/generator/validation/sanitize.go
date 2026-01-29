package validation

import (
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

func SanitizeLabelName(name string) string {
	return strutil.SanitizeLabelName(name)
}
