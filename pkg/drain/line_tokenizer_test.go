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
			tokenizer: &defaultTokenizer{},
			input:     "1234567890",
			expected:  []string{"1234567890", "<END>"},
		},
		{
			name:      "single word",
			tokenizer: &defaultTokenizer{},
			input:     "apple",
			expected:  []string{"apple", "<END>"},
		},
		{
			name:      "word with number",
			tokenizer: &defaultTokenizer{},
			input:     "application_1234567890",
			expected:  []string{"application", "_", "1234567890", "<END>"},
		},
		{
			name:      "SQL LDAP",
			tokenizer: &defaultTokenizer{},
			input:     "INSERT pineapple.mango,cn=org,dc=place,dc=city",
			expected: []string{
				"INSERT", " ", "pineapple", ".",
				"mango", ",", "cn", "=", "org", ",", "dc", "=", "place", ",", "dc", "=", "city",
				"<END>",
			},
		},
		{
			name:      "simple sql",
			tokenizer: &defaultTokenizer{},
			input:     "CREATE TABLE apple (fig)",
			expected: []string{
				"CREATE", " ", "TABLE", " ",
				"apple", " ", "(", "fig", ")",
				"<END>",
			},
		},
		{
			name:      "words with uuid",
			tokenizer: &defaultTokenizer{},
			input:     "TRAMPOLINE COMMAND GO: commence jumping: 6cae3d57-f354-4d14-8420-1a2961327100",
			expected: []string{
				"TRAMPOLINE", " ", "COMMAND", " ",
				"GO", ":", " ", "commence", " ", "jumping", ":", " ",
				"6cae3d57-f354-4d14-8420-1a2961327100",
				"<END>",
			},
		},
		{
			name:      "query with params",
			tokenizer: &defaultTokenizer{},
			input:     "fetch GET http://api.namespace.svc.cluster.local/api/apps/shared/inband/service.method?input=abc%3Dmango%26def%3Dkiwi%26height%3D395",
			expected: []string{
				"fetch", " ", "GET", " ",
				"http", ":", "/", "/", "api", ".", "namespace", ".", "svc", ".", "cluster", ".", "local", "/", "api", "/", "apps", "/", "shared", "/", "inband", "/",
				"service", ".", "method", "?", "input", "=", "abc", "%3D", "mango", "%26", "def", "%3D", "kiwi", "%26", "height", "%3D", "395",
				"<END>",
			},
		},
		{
			name:      "url with hash",
			tokenizer: &defaultTokenizer{},
			input:     "DELETE /apple-banana-kiwi-date-lemon-elderberry-apple?h=b274be36aa1bf3ef7189d41d8cd98f00b204e9800998ecf8427e",
			expected: []string{
				"DELETE", " ", "/", "apple", "-", "banana", "-", "kiwi", "-", "date", "-", "lemon", "-", "elderberry", "-", "apple", "?", "h", "=", "b274be36aa1bf3ef7189d41d8cd98f00b204e9800998ecf8427e",
				"<END>",
			},
		},
		{
			name:      "multi-byte characters chinese",
			tokenizer: &defaultTokenizer{},
			input:     "GET /api/用户/123",
			expected: []string{
				"GET", " ", "/", "api", "/", "用户", "/", "123",
				"<END>",
			},
		},
		{
			name:      "multi-byte characters japanese",
			tokenizer: &defaultTokenizer{},
			input:     "処理 completed タスク456",
			expected: []string{
				"処理", " ", "completed", " ", "タスク456",
				"<END>",
			},
		},
		{
			name:      "emoji in span name",
			tokenizer: &defaultTokenizer{},
			input:     "🚀 deploy service-abc",
			expected: []string{
				"🚀", " ", "deploy", " ", "service", "-", "abc",
				"<END>",
			},
		},
		{
			name:      "mixed emoji and text",
			tokenizer: &defaultTokenizer{},
			input:     "task✅done/item🔥hot",
			expected: []string{
				"task", "✅", "done", "/", "item", "🔥", "hot",
				"<END>",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tokens := test.tokenizer.Tokenize(test.input, nil)
			require.Equal(t, test.expected, tokens)
		})
	}
}
