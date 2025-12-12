package drain

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"
)

type LineTokenizer interface {
	Tokenize(line string, tokens []string, state any, linesDropped *prometheus.CounterVec) ([]string, any)
	Join(tokens []string, state any) string
	Clone(tokens []string, state any) ([]string, any)
}

type PunctuationAndSuffixAwareTokenizer struct{}

var _ LineTokenizer = (*PunctuationAndSuffixAwareTokenizer)(nil)

func (t *PunctuationAndSuffixAwareTokenizer) Tokenize(str string, tokens []string, state interface{}, _ *prometheus.CounterVec) ([]string, interface{}) {
	tokens = tokens[:0]
	var curr string
	currStart := 0
	for i, c := range str {
		switch c {
		case ' ', '/', '.', '_', '@':
			// Encountered a separator.
			// Split and append current buffer if needed.
			if len(curr) > 0 {
				tokens = t.splitSuffix(tokens, curr)
			}
			// Add the separator
			tokens = append(tokens, str[i:i+utf8.RuneLen(c)])
			// Reset the buffer.
			curr = ""
			currStart = i + 1
		default:
			// Keep collecting into the current buffer.
			// RuneLen to account for multibyte characters.
			curr = str[currStart : i+utf8.RuneLen(c)]
		}
	}
	if len(curr) > 0 {
		tokens = t.splitSuffix(tokens, curr)
	}

	// Always add leaf token at the end that prevents turning the real last
	// leaf into a wildcard unnecessarily.
	tokens = append(tokens, "<END>")
	return tokens, nil
}

func (t *PunctuationAndSuffixAwareTokenizer) splitSuffix(tokens []string, token string) []string {
	firstNumberFromLeft := -1
	lastNumberFromRight := -1
	for i, c := range token {
		if unicode.IsNumber(c) {
			firstNumberFromLeft = i
			break
		}
	}
	for i := len(token) - 1; i >= 0; i-- {
		if unicode.IsNumber(rune(token[i])) {
			lastNumberFromRight = i
		} else {
			break
		}
	}
	if firstNumberFromLeft > 0 && lastNumberFromRight > 0 && firstNumberFromLeft == lastNumberFromRight {
		// Single substring of consecutive numbers detected at the end, break it in pieces.
		tokens = append(tokens, token[:firstNumberFromLeft])
		tokens = append(tokens, token[firstNumberFromLeft:])
	} else {
		// If numbers mix throughout, or the string is entirely numbers, then leave as one piece and substitute using normal logic.
		tokens = append(tokens, token)
	}
	return tokens
}

func (t *PunctuationAndSuffixAwareTokenizer) Clone(tokens []string, state interface{}) ([]string, interface{}) {
	res := make([]string, len(tokens))
	copy(res, tokens)
	return res, nil
}

func (t *PunctuationAndSuffixAwareTokenizer) Join(tokens []string, state interface{}) string {
	// Last token is always <END> so we don't need to include it.
	return strings.Join(tokens[0:len(tokens)-1], "")
}
