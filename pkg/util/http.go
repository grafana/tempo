package util

import (
	"strings"
)

// IsRequestBodyTooLarge returns true if the error is "http: request body too large".
func IsRequestBodyTooLarge(err error) bool {
	return err != nil && strings.Contains(err.Error(), "http: request body too large")
}
