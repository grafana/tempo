package secrets

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextCurlLiteralWord1(t *testing.T) {
	// Successful tokens preserve the unread suffix. Malformed shell syntax
	// must remain rejected; callers do not consume partial words on failure.
	cases := []struct {
		name  string
		input string
		word  string
		rest  string
		ok    bool
	}{
		{name: "empty"},
		{name: "whitespace-only", input: " \t "},
		{name: "literal-at-end", input: "curl", word: "curl", ok: true},
		{name: "literal-with-rest", input: " \tcurl \t--silent", word: "curl", rest: " \t--silent", ok: true},
		{name: "single-quoted", input: "'a b'\tnext", word: "a b", rest: "\tnext", ok: true},
		{name: "double-quoted", input: "\"a b\" next", word: "a b", rest: " next", ok: true},
		{name: "quoted-at-end", input: "\"a b\"", word: "a b", ok: true},
		{name: "empty-quote-with-rest", input: "'' next"},
		{name: "empty-quote-at-end", input: "\"\""},
		{name: "single-quoted-shell-syntax", input: "'$`\\;|&<>\r\n\"' next", word: "$`\\;|&<>\r\n\"", rest: " next", ok: true},
		{name: "double-quoted-shell-syntax", input: "\"';|&<>\r\n\" next", word: "';|&<>\r\n", rest: " next", ok: true},
		{name: "concatenated", input: "pre' mid '\"end\"post next", word: "pre mid endpost", rest: " next", ok: true},
		{name: "empty-concatenation", input: "''\"\" next"},
		{name: "empty-quoted-prefix", input: "''word next", word: "word", rest: " next", ok: true},
		{name: "empty-quoted-suffix", input: "word\"\" next", word: "word", rest: " next", ok: true},
		{name: "escaped-space", input: "a\\ b next", word: "a b", rest: " next", ok: true},
		{name: "escaped-quotes", input: "\\'a\\\" next", word: "'a\"", rest: " next", ok: true},
		{name: "double-quoted-escapes", input: "\"a\\$\\`\\q\" next", word: "a$`q", rest: " next", ok: true},
		{name: "escaped-newline", input: "a\\\nb next", word: "a\nb", rest: " next", ok: true},
		{name: "escaped-separator", input: "a\\;b next", word: "a;b", rest: " next", ok: true},
		{name: "unquoted-expansion", input: "a$b next"},
		{name: "double-quoted-substitution", input: "\"a`b\" next"},
		{name: "separator-after-quote", input: "'a';next"},
		{name: "expansion-after-quote", input: "'a'$b next"},
		{name: "unquoted-newline", input: "a\nb next"},
		{name: "dangling-escape", input: "a\\"},
		{name: "quoted-dangling-escape", input: "\"a\\"},
		{name: "unclosed-single-quote", input: "'a b"},
		{name: "unclosed-double-quote", input: "\"a b"},
		{name: "unclosed-concatenation", input: "a'b"},
		{name: "unclosed-quoted-escape", input: "\"a\\ b"},
		{name: "unicode-whitespace-is-literal", input: "\u00a0é\u2003 next", word: "\u00a0é\u2003", rest: " next", ok: true},
		{name: "invalid-utf8-is-literal", input: "\xffé\xfe\tnext", word: "\xffé\xfe", rest: "\tnext", ok: true},
		{name: "quoted-invalid-utf8", input: "\"\xffé\xfe\" next", word: "\xffé\xfe", rest: " next", ok: true},
		{name: "nul-is-literal", input: "a\x00b next", word: "a\x00b", rest: " next", ok: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			word, rest, ok := nextCurlLiteralWord1(tc.input)
			require.Equal(t, tc.ok, ok)
			if ok {
				require.Equal(t, tc.word, word)
				require.Equal(t, tc.rest, rest)
			}
		})
	}
}
