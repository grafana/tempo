package drain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type LineTokenizer interface {
	Tokenize(line string, tokens []string) []string
	Join(tokens []string) string
}

// defaultTokenizer is the default implementation of the LineTokenizer
// interface. It tokenizes a line into a list of tokens using a simple state
// machine. This is the main point of customization for the DRAIN algorithm, and
// is largely domain-specific.
type defaultTokenizer struct{}

var _ LineTokenizer = (*defaultTokenizer)(nil)

func (t *defaultTokenizer) Tokenize(line string, tokens []string) []string {
	tokens = tokens[:0]

	l := &lexer{
		input: line,
	}
	for !l.eof {
		currentState := lexAny
		for currentState != nil {
			currentState = currentState(l)
		}
		if l.token != "" {
			tokens = append(tokens, l.token)
		}
	}

	// The <END> token is added because the drain algorithm often wildcards the
	// last token, which doesn't work well for inputs like `GET /users/123/data`
	// and `GET /users/123/settings` - we want to preserve `/data` and `/settings`.
	// Adding an extra token helps reduce this behavior.
	tokens = append(tokens, "<END>")

	return tokens
}

// lexer implements a simple state machine for tokenizing a string.
// Inspired by Rob Pike's talk on lexers and parsers: https://www.youtube.com/watch?v=HxaD_trXwRE.
// The code from that talk still exists in the stdlib as text/template/parse/lex.go for reference.
type lexer struct {
	input string // the input string to tokenize
	pos   int    // the current position in the input string
	start int    // the start position of the current token
	eof   bool   // whether the end of the input has been reached

	token string
}

// emit emits the current token and resets the start position.
func (l *lexer) emit() stateFn {
	l.token = l.input[l.start:l.pos]
	l.start = l.pos
	return nil
}

// next consumes the next rune from the input and advances the position.
func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.eof = true
		return 0
	}

	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += size
	return r
}

// backup steps back one rune and resets the start position.
func (l *lexer) backup() {
	if !l.eof && l.pos > 0 {
		_, w := utf8.DecodeLastRuneInString(l.input[:l.pos])
		l.pos -= w
	}
}

type stateFn func(*lexer) stateFn

// lexAny is the default state function that matches any rune.
// it branches to special cases for the supported token types.
func lexAny(l *lexer) stateFn {
	l.token = ""
	r := l.next()
	switch {
	case l.eof: // eof
		return nil
	case r == '%':
		return lexURLEncoded
	case unicode.IsSpace(r):
		return lexSpace
	case unicode.Is(unicode.Hex_Digit, r):
		return maybeHex
	case unicode.IsLetter(r) || unicode.IsNumber(r):
		return lexAlphaNumeric
	default:
		// TODO: Consider combining consecutive delimiters (slashes, commas, etc.)
		// into a single token for better pattern matching efficiency.
		return l.emit()
	}
}

// lexURLEncoded is a state function that matches hex encoded characters.
// e.g. `%3D` as might be part of a URL encoded query parameter.
func lexURLEncoded(l *lexer) stateFn {
	// We've already consumed the `%`
	r1 := l.next()
	r2 := l.next()
	if !unicode.Is(unicode.Hex_Digit, r1) || !unicode.Is(unicode.Hex_Digit, r2) {
		// Not a valid hex encoded character, so we emit just the `%` and
		// backtrack to just after it.
		l.backup()
		l.backup()
		return l.emit()
	}

	return l.emit()
}

// maybeHex is a state function that greedily consumes hex digits, but may
// branch to a UUID or alpha numeric if necessary.
func maybeHex(l *lexer) stateFn {
	for {
		r := l.next()
		if unicode.Is(unicode.Hex_Digit, r) {
			continue
		}
		if r == '-' {
			return lexUUID
		}

		// We might have been a little greedy assuming it was hex, but it might still be alpha numeric.
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return lexAlphaNumeric
		}

		break
	}
	l.backup() // don't emit the non-hex
	return l.emit()
}

// lexUUID is a state function that consumes hex and '-' characters.
func lexUUID(l *lexer) stateFn {
	for {
		r := l.next()
		if unicode.Is(unicode.Hex_Digit, r) || r == '-' {
			continue
		}
		break
	}
	l.backup() // don't emit the non-hex
	return l.emit()
}

// lexSpace is a state function that consumes consecutive spaces as a single token.
func lexSpace(l *lexer) stateFn {
	for unicode.IsSpace(l.next()) {
	}
	l.backup() // don't emit the non-space
	return l.emit()
}

// lexAlphaNumeric is a state function that consumes alpha numeric characters.
func lexAlphaNumeric(l *lexer) stateFn {
	for {
		r := l.next()
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			continue
		}
		break
	}
	l.backup() // don't emit the non-alpha numeric
	return l.emit()
}

func (t *defaultTokenizer) Join(tokens []string) string {
	// Last token is always <END> so we don't need to include it.
	return strings.Join(tokens[0:len(tokens)-1], "")
}
