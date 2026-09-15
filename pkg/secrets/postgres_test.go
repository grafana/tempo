package secrets

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostgresConnectionURICandidateLimit(t *testing.T) {
	policy, err := testPolicyCompiler.CompilePolicy(Policy{})
	require.NoError(t, err)
	const prefix = "postgresql://u:p@db.invalid/"
	atLimit := prefix + strings.Repeat("d", maxStructuredCredentialBytes-len(prefix))
	matched := Verdict{Matches: []Match{{RuleID: "postgres-connection-uri"}}}
	for _, test := range []struct {
		name  string
		value string
		want  Verdict
	}{
		{"complete-at-limit", atLimit, matched},
		{"over-limit-is-not-a-prefix-match", atLimit + "d", Verdict{}},
		{"malformed-suffix-is-not-truncated", atLimit + "%GG", Verdict{}},
		{"later-uri-after-oversized-candidate", atLimit + "d " + prefix, matched},
		{"later-uri-after-malformed-candidate", prefix + "%GG " + prefix, matched},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, policy.Detect(test.value))
			batch := policy.NewBatchDetector()
			require.Equal(t, test.want, batch.Detect(test.value))
		})
	}
}
