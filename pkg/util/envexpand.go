package util

import (
	"fmt"
	"os"
	"strings"
)

// ExpandEnv replaces $VAR and ${VAR} references in s with the values of the
// corresponding environment variables. Unsupported POSIX shell parameter
// expansion (e.g. ${VAR:-default}, ${VAR:?error}, ${VAR^^}) is rejected with
// an error so misconfigurations fail at startup rather than silently
// expanding to empty strings.
func ExpandEnv(s string) (string, error) {
	if err := validateExpansions(s); err != nil {
		return "", err
	}
	return os.ExpandEnv(s), nil
}

func validateExpansions(s string) error {
	for i := 0; i < len(s); i++ {
		if s[i] != '$' || i+1 >= len(s) {
			continue
		}
		if s[i+1] == '$' {
			return fmt.Errorf("$$ escape at position %d is not supported; literal $ characters cannot be embedded in env-expanded config values", i)
		}
		if s[i+1] != '{' {
			continue
		}
		end := strings.IndexByte(s[i+2:], '}')
		if end < 0 {
			return fmt.Errorf("unclosed ${...} expression at position %d", i)
		}
		name := s[i+2 : i+2+end]
		for _, c := range name {
			if c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				continue
			}
			return fmt.Errorf("unsupported environment variable expansion %q at position %d: only $VAR and ${VAR} are supported", "${"+name+"}", i)
		}
		i += 2 + end
	}
	return nil
}
