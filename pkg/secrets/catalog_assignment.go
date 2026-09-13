package secrets

import "strings"

// validateAuditedAssignmentContext rejects expression continuations when an
// opaque credential is captured from an assignment or header. It does not
// reinterpret standalone prefixed credentials or complete parsed carriers.
func validateAuditedAssignmentContext(value string, matchStart, matchEnd int, secret string) contextValidation {
	relative := strings.LastIndex(value[matchStart:matchEnd], secret)
	if relative < 0 || secret == "" {
		return contextValidation{}
	}
	start := matchStart + relative
	end := start + len(secret)
	if !strings.ContainsAny(value[matchStart:start], "=:") {
		return contextValidation{accepted: true}
	}
	// Some source-specific validators capture the complete quoted literal.
	quotedCapture := len(secret) >= 2 && (secret[0] == '\'' || secret[0] == '"') && secret[len(secret)-1] == secret[0]
	if !quotedCapture && start > matchStart && (value[start-1] == '\'' || value[start-1] == '"') {
		if end >= len(value) || value[end] != value[start-1] {
			return contextValidation{}
		}
		end++
		quotedCapture = true
	}
	for end < len(value) && (value[end] == ' ' || value[end] == '\t') {
		end++
	}
	if end < len(value) && (strings.ContainsRune("([{.$+-*/%?:=!<>|&\\", rune(value[end])) ||
		quotedCapture && strings.ContainsRune("'\"`", rune(value[end]))) {
		return contextValidation{}
	}
	return contextValidation{accepted: true}
}

// withAuditedAssignmentContext preserves a rule's existing structural checks.
func withAuditedAssignmentContext(next func(string, int, int, string) contextValidation) func(string, int, int, string) contextValidation {
	return func(value string, start, end int, secret string) contextValidation {
		if result := validateAuditedAssignmentContext(value, start, end, secret); !result.accepted {
			return result
		}
		return next(value, start, end, secret)
	}
}
