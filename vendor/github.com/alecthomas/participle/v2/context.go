package participle

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

type contextFieldSet struct {
	tokens     []lexer.Token
	strct      reflect.Value
	field      structLexerField
	fieldValue []reflect.Value
}

// Context for a single parse.
type parseContext struct {
	*lexer.PeekingLexer
	depth             int
	trace             io.Writer
	deepestError      error
	deepestErrorDepth int
	lookahead         int
	caseInsensitive   map[lexer.TokenType]bool
	apply             []*contextFieldSet
	allowTrailing     bool
}

func newParseContext(lex *lexer.PeekingLexer, lookahead int, caseInsensitive map[lexer.TokenType]bool) *parseContext {
	return &parseContext{
		PeekingLexer:    lex,
		caseInsensitive: caseInsensitive,
		lookahead:       lookahead,
	}
}

func (p *parseContext) DeepestError(err error) error {
	if p.PeekingLexer.Cursor() >= p.deepestErrorDepth {
		return err
	}
	if p.deepestError != nil {
		return p.deepestError
	}
	return err
}

// Defer adds a function to be applied once a branch has been picked.
func (p *parseContext) Defer(tokens []lexer.Token, strct reflect.Value, field structLexerField, fieldValue []reflect.Value) {
	p.apply = append(p.apply, &contextFieldSet{tokens, strct, field, fieldValue})
}

// Apply deferred functions.
func (p *parseContext) Apply() error {
	for _, apply := range p.apply {
		if err := setField(apply.tokens, apply.strct, apply.field, apply.fieldValue); err != nil {
			return err
		}
	}
	p.apply = nil
	return nil
}

// Branch accepts the branch as the correct branch.
func (p *parseContext) Accept(branch *parseContext) {
	p.apply = append(p.apply, branch.apply...)
	p.PeekingLexer = branch.PeekingLexer
	if branch.deepestErrorDepth >= p.deepestErrorDepth {
		p.deepestErrorDepth = branch.deepestErrorDepth
		p.deepestError = branch.deepestError
	}
}

// Branch starts a new lookahead branch.
func (p *parseContext) Branch() *parseContext {
	branch := &parseContext{}
	*branch = *p
	branch.apply = nil
	branch.PeekingLexer = p.PeekingLexer.Clone()
	return branch
}

func (p *parseContext) MaybeUpdateError(err error) {
	if p.PeekingLexer.Cursor() >= p.deepestErrorDepth {
		p.deepestError = err
		p.deepestErrorDepth = p.PeekingLexer.Cursor()
	}
}

// Stop returns true if parsing should terminate after the given "branch" failed to match.
//
// Additionally, "err" should be the branch error, if any. This will be tracked to
// aid in error reporting under the assumption that the deepest occurring error is more
// useful than errors further up.
func (p *parseContext) Stop(err error, branch *parseContext) bool {
	if branch.PeekingLexer.Cursor() >= p.deepestErrorDepth {
		p.deepestError = err
		p.deepestErrorDepth = maxInt(branch.PeekingLexer.Cursor(), branch.deepestErrorDepth)
	}
	if !p.hasInfiniteLookahead() && branch.PeekingLexer.Cursor() > p.PeekingLexer.Cursor()+p.lookahead {
		p.Accept(branch)
		return true
	}
	return false
}

func (p *parseContext) hasInfiniteLookahead() bool { return p.lookahead < 0 }

func (p *parseContext) printTrace(n node) func() {
	if p.trace != nil {
		tok := p.PeekingLexer.Peek()
		fmt.Fprintf(p.trace, "%s%q %s\n", strings.Repeat(" ", p.depth*2), tok, n.GoString())
		p.depth += 1
		return func() { p.depth -= 1 }
	}
	return func() {}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
