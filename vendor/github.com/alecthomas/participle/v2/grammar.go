package participle

import (
	"fmt"
	"reflect"
	"text/scanner"

	"github.com/alecthomas/participle/v2/lexer"
)

type generatorContext struct {
	lexer.Definition
	typeNodes    map[reflect.Type]node
	symbolsToIDs map[lexer.TokenType]string
}

func newGeneratorContext(lex lexer.Definition) *generatorContext {
	return &generatorContext{
		Definition:   lex,
		typeNodes:    map[reflect.Type]node{},
		symbolsToIDs: lexer.SymbolsByRune(lex),
	}
}

func (g *generatorContext) addUnionDefs(defs []unionDef) error {
	unionNodes := make([]*union, len(defs))
	for i, def := range defs {
		if _, exists := g.typeNodes[def.typ]; exists {
			return fmt.Errorf("duplicate definition for interface or union type %s", def.typ)
		}
		unionNode := &union{
			unionDef:    def,
			disjunction: disjunction{nodes: make([]node, 0, len(def.members))},
		}
		g.typeNodes[def.typ], unionNodes[i] = unionNode, unionNode
	}
	for i, def := range defs {
		unionNode := unionNodes[i]
		for _, memberType := range def.members {
			memberNode, err := g.parseType(memberType)
			if err != nil {
				return err
			}
			unionNode.disjunction.nodes = append(unionNode.disjunction.nodes, memberNode)
		}
	}
	return nil
}

func (g *generatorContext) addCustomDefs(defs []customDef) error {
	for _, def := range defs {
		if _, exists := g.typeNodes[def.typ]; exists {
			return fmt.Errorf("duplicate definition for interface or union type %s", def.typ)
		}
		g.typeNodes[def.typ] = &custom{typ: def.typ, parseFn: def.parseFn}
	}
	return nil
}

// Takes a type and builds a tree of nodes out of it.
func (g *generatorContext) parseType(t reflect.Type) (_ node, returnedError error) {
	t = indirectType(t)
	if n, ok := g.typeNodes[t]; ok {
		if s, ok := n.(*strct); ok {
			s.usages++
		}
		return n, nil
	}
	if t.Implements(parseableType) {
		return &parseable{t.Elem()}, nil
	}
	if reflect.PtrTo(t).Implements(parseableType) {
		return &parseable{t}, nil
	}
	switch t.Kind() { // nolint: exhaustive
	case reflect.Slice, reflect.Ptr:
		t = indirectType(t.Elem())
		if t.Kind() != reflect.Struct {
			return nil, fmt.Errorf("expected a struct but got %T", t)
		}
		fallthrough

	case reflect.Struct:
		slexer, err := lexStruct(t)
		if err != nil {
			return nil, err
		}
		out := newStrct(t)
		g.typeNodes[t] = out // Ensure we avoid infinite recursion.
		if slexer.NumField() == 0 {
			return nil, fmt.Errorf("can not parse into empty struct %s", t)
		}
		defer decorate(&returnedError, func() string { return slexer.Field().Name })
		e, err := g.parseDisjunction(slexer)
		if err != nil {
			return nil, err
		}
		if e == nil {
			return nil, fmt.Errorf("no grammar found in %s", t)
		}
		if token, _ := slexer.Peek(); !token.EOF() {
			return nil, fmt.Errorf("unexpected input %q", token.Value)
		}
		out.expr = e
		return out, nil
	}
	return nil, fmt.Errorf("%s should be a struct or should implement the Parseable interface", t)
}

func (g *generatorContext) parseDisjunction(slexer *structLexer) (node, error) {
	out := &disjunction{}
	for {
		n, err := g.parseSequence(slexer)
		if err != nil {
			return nil, err
		}
		if n == nil {
			return nil, fmt.Errorf("alternative expression %d cannot be empty", len(out.nodes)+1)
		}
		out.nodes = append(out.nodes, n)
		if token, _ := slexer.Peek(); token.Type != '|' {
			break
		}
		_, err = slexer.Next() // |
		if err != nil {
			return nil, err
		}
	}
	if len(out.nodes) == 1 {
		return out.nodes[0], nil
	}
	return out, nil
}

func (g *generatorContext) parseSequence(slexer *structLexer) (node, error) {
	head := &sequence{}
	cursor := head
loop:
	for {
		if token, err := slexer.Peek(); err != nil {
			return nil, err
		} else if token.Type == lexer.EOF {
			break loop
		}
		term, err := g.parseTerm(slexer, true)
		if err != nil {
			return nil, err
		}
		if term == nil {
			break loop
		}
		if cursor.node == nil {
			cursor.head = true
			cursor.node = term
		} else {
			cursor.next = &sequence{node: term}
			cursor = cursor.next
		}
	}
	if head.node == nil {
		return nil, nil
	}
	if head.next == nil {
		return head.node, nil
	}
	return head, nil
}

func (g *generatorContext) parseTermNoModifiers(slexer *structLexer, allowUnknown bool) (node, error) {
	t, err := slexer.Peek()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case '@':
		return g.parseCapture(slexer)
	case scanner.String, scanner.RawString, scanner.Char:
		return g.parseLiteral(slexer)
	case '!', '~':
		return g.parseNegation(slexer)
	case '[':
		return g.parseOptional(slexer)
	case '{':
		return g.parseRepetition(slexer)
	case '(':
		// Also handles (? used for lookahead groups
		return g.parseGroup(slexer)
	case scanner.Ident:
		return g.parseReference(slexer)
	case lexer.EOF:
		_, _ = slexer.Next()
		return nil, nil
	default:
		if allowUnknown {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected token %v", t)
	}
}

func (g *generatorContext) parseTerm(slexer *structLexer, allowUnknown bool) (node, error) {
	out, err := g.parseTermNoModifiers(slexer, allowUnknown)
	if err != nil {
		return nil, err
	}
	return g.parseModifier(slexer, out)
}

// Parse modifiers: ?, *, + and/or !
func (g *generatorContext) parseModifier(slexer *structLexer, expr node) (node, error) {
	out := &group{expr: expr}
	t, err := slexer.Peek()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case '!':
		out.mode = groupMatchNonEmpty
	case '+':
		out.mode = groupMatchOneOrMore
	case '*':
		out.mode = groupMatchZeroOrMore
	case '?':
		out.mode = groupMatchZeroOrOne
	default:
		return expr, nil
	}
	_, _ = slexer.Next()
	return out, nil
}

// @<expression> captures <expression> into the current field.
func (g *generatorContext) parseCapture(slexer *structLexer) (node, error) {
	_, _ = slexer.Next()
	token, err := slexer.Peek()
	if err != nil {
		return nil, err
	}
	field := slexer.Field()
	if token.Type == '@' {
		_, _ = slexer.Next()
		n, err := g.parseType(field.Type)
		if err != nil {
			return nil, err
		}
		return &capture{field, n}, nil
	}
	ft := indirectType(field.Type)
	if ft.Kind() == reflect.Struct && ft != tokenType && ft != tokensType && !implements(ft, captureType) && !implements(ft, textUnmarshalerType) {
		return nil, fmt.Errorf("%s: structs can only be parsed with @@ or by implementing the Capture or encoding.TextUnmarshaler interfaces", ft)
	}
	n, err := g.parseTermNoModifiers(slexer, false)
	if err != nil {
		return nil, err
	}
	return &capture{field, n}, nil
}

// A reference in the form <identifier> refers to a named token from the lexer.
func (g *generatorContext) parseReference(slexer *structLexer) (node, error) { // nolint: interfacer
	token, err := slexer.Next()
	if err != nil {
		return nil, err
	}
	if token.Type != scanner.Ident {
		return nil, fmt.Errorf("expected identifier but got %q", token)
	}
	typ, ok := g.Symbols()[token.Value]
	if !ok {
		return nil, fmt.Errorf("unknown token type %q", token)
	}
	return &reference{typ: typ, identifier: token.Value}, nil
}

// [ <expression> ] optionally matches <expression>.
func (g *generatorContext) parseOptional(slexer *structLexer) (node, error) {
	_, _ = slexer.Next() // [
	disj, err := g.parseDisjunction(slexer)
	if err != nil {
		return nil, err
	}
	n := &group{expr: disj, mode: groupMatchZeroOrOne}
	next, err := slexer.Next()
	if err != nil {
		return nil, err
	}
	if next.Type != ']' {
		return nil, fmt.Errorf("expected ] but got %q", next)
	}
	return n, nil
}

// { <expression> } matches 0 or more repititions of <expression>
func (g *generatorContext) parseRepetition(slexer *structLexer) (node, error) {
	_, _ = slexer.Next() // {
	disj, err := g.parseDisjunction(slexer)
	if err != nil {
		return nil, err
	}
	n := &group{expr: disj, mode: groupMatchZeroOrMore}
	next, err := slexer.Next()
	if err != nil {
		return nil, err
	}
	if next.Type != '}' {
		return nil, fmt.Errorf("expected } but got %q", next)
	}
	return n, nil
}

// ( <expression> ) groups a sub-expression
func (g *generatorContext) parseGroup(slexer *structLexer) (node, error) {
	_, _ = slexer.Next() // (
	peek, err := slexer.Peek()
	if err != nil {
		return nil, err
	}
	if peek.Type == '?' {
		return g.subparseLookaheadGroup(slexer) // If there was an error peeking, code below will handle it
	}
	expr, err := g.subparseGroup(slexer)
	if err != nil {
		return nil, err
	}
	return &group{expr: expr}, nil
}

// (?[!=] <expression> ) requires a grouped sub-expression either matches or doesn't match, without consuming it
func (g *generatorContext) subparseLookaheadGroup(slexer *structLexer) (node, error) {
	_, _ = slexer.Next() // ? - the opening ( was already consumed in parseGroup
	var negative bool
	next, err := slexer.Next()
	if err != nil {
		return nil, err
	}
	switch next.Type {
	case '=':
		negative = false
	case '!':
		negative = true
	default:
		return nil, fmt.Errorf("expected = or ! but got %q", next)
	}
	expr, err := g.subparseGroup(slexer)
	if err != nil {
		return nil, err
	}
	return &lookaheadGroup{expr: expr, negative: negative}, nil
}

// helper parsing <expression> ) to finish parsing groups or lookahead groups
func (g *generatorContext) subparseGroup(slexer *structLexer) (node, error) {
	disj, err := g.parseDisjunction(slexer)
	if err != nil {
		return nil, err
	}
	next, err := slexer.Next() // )
	if err != nil {
		return nil, err
	}
	if next.Type != ')' {
		return nil, fmt.Errorf("expected ) but got %q", next)
	}
	return disj, nil
}

// A token negation
//
// Accepts both the form !"some-literal" and !SomeNamedToken
func (g *generatorContext) parseNegation(slexer *structLexer) (node, error) {
	_, _ = slexer.Next() // advance the parser since we have '!' right now.
	next, err := g.parseTermNoModifiers(slexer, false)
	if err != nil {
		return nil, err
	}
	return &negation{next}, nil
}

// A literal string.
//
// Note that for this to match, the tokeniser must be able to produce this string. For example,
// if the tokeniser only produces individual characters but the literal is "hello", or vice versa.
func (g *generatorContext) parseLiteral(lex *structLexer) (node, error) { // nolint: interfacer
	token, err := lex.Next()
	if err != nil {
		return nil, err
	}
	s := token.Value
	t := lexer.TokenType(-1)
	token, err = lex.Peek()
	if err != nil {
		return nil, err
	}
	if token.Type == ':' {
		_, _ = lex.Next()
		token, err = lex.Next()
		if err != nil {
			return nil, err
		}
		if token.Type != scanner.Ident {
			return nil, fmt.Errorf("expected identifier for literal type constraint but got %q", token)
		}
		var ok bool
		t, ok = g.Symbols()[token.Value]
		if !ok {
			return nil, fmt.Errorf("unknown token type %q in literal type constraint", token)
		}
	}
	return &literal{s: s, t: t, tt: g.symbolsToIDs[t]}, nil
}

func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		return indirectType(t.Elem())
	}
	return t
}

func implements(t, i reflect.Type) bool {
	return t.Implements(i) || reflect.PtrTo(t).Implements(i)
}
