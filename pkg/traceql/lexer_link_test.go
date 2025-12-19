package traceql

import (
	"strings"
	"testing"
	"text/scanner"

	"github.com/stretchr/testify/require"
)

func TestLexerLinkTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"->>", LINK_TO},
		{"<<-", LINK_FROM},
		{"&->>", UNION_LINK_TO},
		{"&<<-", UNION_LINK_FROM},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer{
				Scanner: scanner.Scanner{
					Mode: scanner.SkipComments | scanner.ScanStrings,
				},
			}
			l.Init(strings.NewReader(tt.input))
			
			lval := &yySymType{}
			tok := l.Lex(lval)
			
			require.Equal(t, tt.expected, tok, "expected token %d (%s), got %d", tt.expected, tt.input, tok)
		})
	}
}

