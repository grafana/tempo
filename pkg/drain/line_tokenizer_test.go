package drain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLineTokenizer(t *testing.T) {
	tests := []struct {
		name      string
		tokenizer LineTokenizer
		input     string
		expected  []string
	}{
		{
			name:      "single number",
			tokenizer: &PunctuationAndSuffixAwareTokenizer{},
			input:     "1234567890",
			expected:  []string{"1234567890", "<END>"},
		},
		{
			name:      "single word",
			tokenizer: &PunctuationAndSuffixAwareTokenizer{},
			input:     "apple",
			expected:  []string{"apple", "<END>"},
		},
		{
			name:      "word with number",
			tokenizer: &PunctuationAndSuffixAwareTokenizer{},
			input:     "application_1234567890",
			expected:  []string{"application", "_", "1234567890", "<END>"},
		},
		{
			name:      "SQL LDAP",
			tokenizer: &PunctuationAndSuffixAwareTokenizer{},
			input:     "INSERT pineapple.mango,cn=org,dc=place,dc=city",
			expected:  []string{"INSERT", " ", "pineapple", ".", "mango,cn=org,dc=place,dc=city", "<END>"},
		},
		{
			name:      "words with uuid",
			tokenizer: &PunctuationAndSuffixAwareTokenizer{},
			input:     "TRAMPOLINE COMMAND GO: commence jumping: 6cae3d57-f354-4d14-8420-1a2961327100",
			expected:  []string{"TRAMPOLINE", " ", "COMMAND", " ", "GO:", " ", "commence", " ", "jumping:", " ", "6cae3d57-f354-4d14-8420-1a2961327100", "<END>"},
		},
		{
			name:      "query with params",
			tokenizer: &PunctuationAndSuffixAwareTokenizer{},
			input:     "fetch GET http://api.namespace.svc.cluster.local/api/apps/shared/inband/service.method?input=abc%3Dmango%26def%3Dkiwi%26height%3D395",
			expected: []string{
				"fetch", " ", "GET", " ",
				"http:", "/", "/", "api", ".", "namespace", ".", "svc", ".", "cluster", ".", "local", "/", "api", "/", "apps", "/", "shared", "/", "inband", "/",
				"service", ".", "method?input=abc%3Dmango%26def%3Dkiwi%26height%3D395",
				"<END>"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tokens, state := test.tokenizer.Tokenize(test.input, nil, nil, nil)
			require.Equal(t, test.expected, tokens)
			require.Nil(t, state)
		})
	}
}
