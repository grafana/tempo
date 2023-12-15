package participle

import (
	"fmt"
	"io"
	"reflect"

	"github.com/alecthomas/participle/v2/lexer"
)

// MaxLookahead can be used with UseLookahead to get pseudo-infinite
// lookahead without the risk of pathological cases causing a stack
// overflow.
const MaxLookahead = 99999

// An Option to modify the behaviour of the Parser.
type Option func(p *parserOptions) error

// Lexer is an Option that sets the lexer to use with the given grammar.
func Lexer(def lexer.Definition) Option {
	return func(p *parserOptions) error {
		p.lex = def
		return nil
	}
}

// UseLookahead allows branch lookahead up to "n" tokens.
//
// If parsing cannot be disambiguated before "n" tokens of lookahead, parsing will fail.
//
// Note that increasing lookahead has a minor performance impact, but also
// reduces the accuracy of error reporting.
//
// If "n" is negative, it will be treated as "infinite" lookahead.
// This can have a large impact on performance, and does not provide any
// protection against stack overflow during parsing.
// In most cases, using MaxLookahead will achieve the same results in practice,
// but with a concrete upper bound to prevent pathological behavior in the parser.
// Using infinite lookahead can be useful for testing, or for parsing especially
// ambiguous grammars. Use at your own risk!
func UseLookahead(n int) Option {
	return func(p *parserOptions) error {
		p.useLookahead = n
		return nil
	}
}

// CaseInsensitive allows the specified token types to be matched case-insensitively.
//
// Note that the lexer itself will also have to be case-insensitive; this option
// just controls whether literals in the grammar are matched case insensitively.
func CaseInsensitive(tokens ...string) Option {
	return func(p *parserOptions) error {
		for _, token := range tokens {
			p.caseInsensitive[token] = true
		}
		return nil
	}
}

// ParseTypeWith associates a custom parsing function with some interface type T.
// When the parser encounters a value of type T, it will use the given parse function to
// parse a value from the input.
//
// The parse function may return anything it wishes as long as that value satisfies the interface T.
// However, only a single function can be defined for any type T.
// If you want to have multiple parse functions returning types that satisfy the same interface, you'll
// need to define new wrapper types for each one.
//
// This can be useful if you want to parse a DSL within the larger grammar, or if you want
// to implement an optimized parsing scheme for some portion of the grammar.
func ParseTypeWith[T any](parseFn func(*lexer.PeekingLexer) (T, error)) Option {
	return func(p *parserOptions) error {
		parseFnVal := reflect.ValueOf(parseFn)
		parseFnType := parseFnVal.Type()
		if parseFnType.Out(0).Kind() != reflect.Interface {
			return fmt.Errorf("ParseTypeWith: T must be an interface type (got %s)", parseFnType.Out(0))
		}
		prodType := parseFnType.Out(0)
		p.customDefs = append(p.customDefs, customDef{prodType, parseFnVal})
		return nil
	}
}

// Union associates several member productions with some interface type T.
// Given members X, Y, Z, and W for a union type U, then the EBNF rule is:
//
//	U = X | Y | Z | W .
//
// When the parser encounters a field of type T, it will attempt to parse each member
// in sequence and take the first match. Because of this, the order in which the
// members are defined is important. You must be careful to order your members appropriately.
//
// An example of a bad parse that can happen if members are out of order:
//
// If the first member matches A, and the second member matches A B,
// and the source string is "AB", then the parser will only match A, and will not
// try to parse the second member at all.
func Union[T any](members ...T) Option {
	return func(p *parserOptions) error {
		var t T
		unionType := reflect.TypeOf(&t).Elem()
		if unionType.Kind() != reflect.Interface {
			return fmt.Errorf("union: union type must be an interface (got %s)", unionType)
		}
		memberTypes := make([]reflect.Type, 0, len(members))
		for _, m := range members {
			memberTypes = append(memberTypes, reflect.TypeOf(m))
		}
		p.unionDefs = append(p.unionDefs, unionDef{unionType, memberTypes})
		return nil
	}
}

// ParseOption modifies how an individual parse is applied.
type ParseOption func(p *parseContext)

// Trace the parse to "w".
func Trace(w io.Writer) ParseOption {
	return func(p *parseContext) {
		p.trace = w
	}
}

// AllowTrailing tokens without erroring.
//
// That is, do not error if a full parse completes but additional tokens remain.
func AllowTrailing(ok bool) ParseOption {
	return func(p *parseContext) {
		p.allowTrailing = ok
	}
}
