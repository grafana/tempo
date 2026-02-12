package livestore

import (
	"fmt"
	"regexp"
)

// sanitizePanicMessage removes sensitive information from panic messages
// before logging or returning them to callers.
//
// Redactions:
// - Filesystem paths: /path/to/file -> /***
// - Connection strings: user:pass@host -> ***@host
// - API keys/tokens: 32+ char hex strings -> ***
func sanitizePanicMessage(panicValue interface{}) string {
	msg := fmt.Sprintf("%v", panicValue)

	// Redact connection string credentials FIRST (before path redaction)
	// This prevents :// from being split by path matching
	msg = regexp.MustCompile(`://[^@]+@`).ReplaceAllString(msg, "://***@")

	// Redact filesystem paths (but not :// in URLs)
	// Negative lookbehind would be ideal but Go doesn't support it,
	// so we use a more conservative pattern
	msg = regexp.MustCompile(`(?:^|[^:])/[a-zA-Z0-9/_\-.]+`).ReplaceAllStringFunc(msg, func(s string) string {
		if s[0] == ':' {
			return s // Don't redact :// sequences
		}
		// Preserve first character if not ':', then redact path
		if s[0] != '/' {
			return string(s[0]) + "/***"
		}
		return "/***"
	})

	// Redact long hex strings (likely API keys/tokens)
	msg = regexp.MustCompile(`[a-fA-F0-9]{32,}`).ReplaceAllString(msg, "***")

	return msg
}
