package participle

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

type unionDef struct {
	typ     reflect.Type
	members []reflect.Type
}

type customDef struct {
	typ     reflect.Type
	parseFn reflect.Value
}

type parserOptions struct {
	lex                   lexer.Definition
	rootType              reflect.Type
	typeNodes             map[reflect.Type]node
	useLookahead          int
	caseInsensitive       map[string]bool
	caseInsensitiveTokens map[lexer.TokenType]bool
	mappers               []mapperByToken
	unionDefs             []unionDef
	customDefs            []customDef
	elide                 []string
}

// A Parser for a particular grammar and lexer.
type Parser[G any] struct {
	parserOptions
}

// ParserForProduction returns a new parser for the given production in grammar G.
func ParserForProduction[P, G any](parser *Parser[G]) (*Parser[P], error) {
	t := reflect.TypeOf(*new(P))
	_, ok := parser.typeNodes[t]
	if !ok {
		return nil, fmt.Errorf("parser does not contain a production of type %s", t)
	}
	return (*Parser[P])(parser), nil
}

// MustBuild calls Build[G](options...) and panics if an error occurs.
func MustBuild[G any](options ...Option) *Parser[G] {
	parser, err := Build[G](options...)
	if err != nil {
		panic(err)
	}
	return parser
}

// Build constructs a parser for the given grammar.
//
// If "Lexer()" is not provided as an option, a default lexer based on text/scanner will be used. This scans typical Go-
// like tokens.
//
// See documentation for details.
func Build[G any](options ...Option) (parser *Parser[G], err error) {
	// Configure Parser[G] struct with defaults + options.
	p := &Parser[G]{
		parserOptions: parserOptions{
			lex:             lexer.TextScannerLexer,
			caseInsensitive: map[string]bool{},
			useLookahead:    1,
		},
	}
	for _, option := range options {
		if err = option(&p.parserOptions); err != nil {
			return nil, err
		}
	}

	symbols := p.lex.Symbols()
	if len(p.mappers) > 0 {
		mappers := map[lexer.TokenType][]Mapper{}
		for _, mapper := range p.mappers {
			if len(mapper.symbols) == 0 {
				mappers[lexer.EOF] = append(mappers[lexer.EOF], mapper.mapper)
			} else {
				for _, symbol := range mapper.symbols {
					if rn, ok := symbols[symbol]; !ok {
						return nil, fmt.Errorf("mapper %#v uses unknown token %q", mapper, symbol)
					} else { // nolint: golint
						mappers[rn] = append(mappers[rn], mapper.mapper)
					}
				}
			}
		}
		p.lex = &mappingLexerDef{p.lex, func(t lexer.Token) (lexer.Token, error) {
			combined := make([]Mapper, 0, len(mappers[t.Type])+len(mappers[lexer.EOF]))
			combined = append(combined, mappers[lexer.EOF]...)
			combined = append(combined, mappers[t.Type]...)

			var err error
			for _, m := range combined {
				t, err = m(t)
				if err != nil {
					return t, err
				}
			}
			return t, nil
		}}
	}

	context := newGeneratorContext(p.lex)
	if err := context.addCustomDefs(p.customDefs); err != nil {
		return nil, err
	}
	if err := context.addUnionDefs(p.unionDefs); err != nil {
		return nil, err
	}

	var grammar G
	v := reflect.ValueOf(&grammar)
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	p.rootType = v.Type()
	rootNode, err := context.parseType(p.rootType)
	if err != nil {
		return nil, err
	}
	if err := validate(rootNode); err != nil {
		return nil, err
	}
	p.typeNodes = context.typeNodes
	p.typeNodes[p.rootType] = rootNode
	p.setCaseInsensitiveTokens()
	return p, nil
}

// Lexer returns the parser's builtin lexer.
func (p *Parser[G]) Lexer() lexer.Definition {
	return p.lex
}

// Lex uses the parser's lexer to tokenise input.
// Parameter filename is used as an opaque prefix in error messages.
func (p *Parser[G]) Lex(filename string, r io.Reader) ([]lexer.Token, error) {
	lex, err := p.lex.Lex(filename, r)
	if err != nil {
		return nil, err
	}
	tokens, err := lexer.ConsumeAll(lex)
	return tokens, err
}

// ParseFromLexer into grammar v which must be of the same type as the grammar passed to
// Build().
//
// This may return a Error.
func (p *Parser[G]) ParseFromLexer(lex *lexer.PeekingLexer, options ...ParseOption) (*G, error) {
	v := new(G)
	rv := reflect.ValueOf(v)
	parseNode, err := p.parseNodeFor(rv)
	if err != nil {
		return nil, err
	}
	ctx := newParseContext(lex, p.useLookahead, p.caseInsensitiveTokens)
	defer func() { *lex = ctx.PeekingLexer }()
	for _, option := range options {
		option(&ctx)
	}
	// If the grammar implements Parseable, use it.
	if parseable, ok := any(v).(Parseable); ok {
		return v, p.rootParseable(&ctx, parseable)
	}
	return v, p.parseOne(&ctx, parseNode, rv)
}

func (p *Parser[G]) setCaseInsensitiveTokens() {
	p.caseInsensitiveTokens = map[lexer.TokenType]bool{}
	for sym, tt := range p.lex.Symbols() {
		if p.caseInsensitive[sym] {
			p.caseInsensitiveTokens[tt] = true
		}
	}
}

func (p *Parser[G]) parse(lex lexer.Lexer, options ...ParseOption) (v *G, err error) {
	peeker, err := lexer.Upgrade(lex, p.getElidedTypes()...)
	if err != nil {
		return nil, err
	}
	return p.ParseFromLexer(peeker, options...)
}

// Parse from r into grammar v which must be of the same type as the grammar passed to
// Build(). Parameter filename is used as an opaque prefix in error messages.
//
// This may return an Error.
func (p *Parser[G]) Parse(filename string, r io.Reader, options ...ParseOption) (v *G, err error) {
	if filename == "" {
		filename = lexer.NameOfReader(r)
	}
	lex, err := p.lex.Lex(filename, r)
	if err != nil {
		return nil, err
	}
	return p.parse(lex, options...)
}

// ParseString from s into grammar v which must be of the same type as the grammar passed to
// Build(). Parameter filename is used as an opaque prefix in error messages.
//
// This may return an Error.
func (p *Parser[G]) ParseString(filename string, s string, options ...ParseOption) (v *G, err error) {
	var lex lexer.Lexer
	if sl, ok := p.lex.(lexer.StringDefinition); ok {
		lex, err = sl.LexString(filename, s)
	} else {
		lex, err = p.lex.Lex(filename, strings.NewReader(s))
	}
	if err != nil {
		return nil, err
	}
	return p.parse(lex, options...)
}

// ParseBytes from b into grammar v which must be of the same type as the grammar passed to
// Build(). Parameter filename is used as an opaque prefix in error messages.
//
// This may return an Error.
func (p *Parser[G]) ParseBytes(filename string, b []byte, options ...ParseOption) (v *G, err error) {
	var lex lexer.Lexer
	if sl, ok := p.lex.(lexer.BytesDefinition); ok {
		lex, err = sl.LexBytes(filename, b)
	} else {
		lex, err = p.lex.Lex(filename, bytes.NewReader(b))
	}
	if err != nil {
		return nil, err
	}
	return p.parse(lex, options...)
}

func (p *Parser[G]) parseOne(ctx *parseContext, parseNode node, rv reflect.Value) error {
	err := p.parseInto(ctx, parseNode, rv)
	if err != nil {
		return err
	}
	token := ctx.Peek()
	if !token.EOF() && !ctx.allowTrailing {
		return ctx.DeepestError(&UnexpectedTokenError{Unexpected: *token})
	}
	return nil
}

func (p *Parser[G]) parseInto(ctx *parseContext, parseNode node, rv reflect.Value) error {
	if rv.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer to a struct or interface, but is a nil %s", rv.Type())
	}
	pv, err := p.typeNodes[rv.Type().Elem()].Parse(ctx, rv.Elem())
	if len(pv) > 0 && pv[0].Type() == rv.Elem().Type() {
		rv.Elem().Set(reflect.Indirect(pv[0]))
	}
	if err != nil {
		return err
	}
	if pv == nil {
		token := ctx.Peek()
		return ctx.DeepestError(&UnexpectedTokenError{Unexpected: *token})
	}
	return nil
}

func (p *Parser[G]) rootParseable(ctx *parseContext, parseable Parseable) error {
	if err := parseable.Parse(&ctx.PeekingLexer); err != nil {
		if err == NextMatch {
			err = &UnexpectedTokenError{Unexpected: *ctx.Peek()}
		} else {
			err = &ParseError{Msg: err.Error(), Pos: ctx.Peek().Pos}
		}
		return ctx.DeepestError(err)
	}
	peek := ctx.Peek()
	if !peek.EOF() && !ctx.allowTrailing {
		return ctx.DeepestError(&UnexpectedTokenError{Unexpected: *peek})
	}
	return nil
}

func (p *Parser[G]) getElidedTypes() []lexer.TokenType {
	symbols := p.lex.Symbols()
	elideTypes := make([]lexer.TokenType, 0, len(p.elide))
	for _, elide := range p.elide {
		rn, ok := symbols[elide]
		if !ok {
			panic(fmt.Errorf("Elide() uses unknown token %q", elide))
		}
		elideTypes = append(elideTypes, rn)
	}
	return elideTypes
}

func (p *Parser[G]) parseNodeFor(v reflect.Value) (node, error) {
	t := v.Type()
	if t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	if t.Kind() != reflect.Ptr || (t.Elem().Kind() != reflect.Struct && t.Elem().Kind() != reflect.Interface) {
		return nil, fmt.Errorf("expected a pointer to a struct or interface, but got %s", t)
	}
	parseNode := p.typeNodes[t]
	if parseNode == nil {
		t = t.Elem()
		parseNode = p.typeNodes[t]
	}
	if parseNode == nil {
		return nil, fmt.Errorf("parser does not know how to parse values of type %s", t)
	}
	return parseNode, nil
}
