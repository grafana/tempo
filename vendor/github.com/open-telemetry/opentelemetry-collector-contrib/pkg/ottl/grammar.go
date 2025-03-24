// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

// parsedStatement represents a parsed statement. It is the entry point into the statement DSL.
type parsedStatement struct {
	Editor editor `parser:"(@@"`
	// If converter is matched then return error
	Converter   *converter         `parser:"|@@)"`
	WhereClause *booleanExpression `parser:"( 'where' @@ )?"`
}

func (p *parsedStatement) checkForCustomError() error {
	validator := &grammarCustomErrorsVisitor{}
	if p.Converter != nil {
		validator.add(fmt.Errorf("editor names must start with a lowercase letter but got '%v'", p.Converter.Function))
	}

	p.Editor.accept(validator)
	if p.WhereClause != nil {
		p.WhereClause.accept(validator)
	}

	return validator.join()
}

type constExpr struct {
	Boolean   *boolean   `parser:"( @Boolean"`
	Converter *converter `parser:"| @@ )"`
}

// booleanValue represents something that evaluates to a boolean --
// either an equality or inequality, explicit true or false, or
// a parenthesized subexpression.
type booleanValue struct {
	Negation   *string            `parser:"@OpNot?"`
	Comparison *comparison        `parser:"( @@"`
	ConstExpr  *constExpr         `parser:"| @@"`
	SubExpr    *booleanExpression `parser:"| '(' @@ ')' )"`
}

func (b *booleanValue) accept(v grammarVisitor) {
	if b.Comparison != nil {
		b.Comparison.accept(v)
	}
	if b.ConstExpr != nil && b.ConstExpr.Converter != nil {
		b.ConstExpr.Converter.accept(v)
	}
	if b.SubExpr != nil {
		b.SubExpr.accept(v)
	}
}

// opAndBooleanValue represents the right side of an AND boolean expression.
type opAndBooleanValue struct {
	Operator string        `parser:"@OpAnd"`
	Value    *booleanValue `parser:"@@"`
}

func (b *opAndBooleanValue) accept(v grammarVisitor) {
	if b.Value != nil {
		b.Value.accept(v)
	}
}

// term represents an arbitrary number of boolean values joined by AND.
type term struct {
	Left  *booleanValue        `parser:"@@"`
	Right []*opAndBooleanValue `parser:"@@*"`
}

func (b *term) accept(v grammarVisitor) {
	if b.Left != nil {
		b.Left.accept(v)
	}
	for _, r := range b.Right {
		if r != nil {
			r.accept(v)
		}
	}
}

// opOrTerm represents the right side of an OR boolean expression.
type opOrTerm struct {
	Operator string `parser:"@OpOr"`
	Term     *term  `parser:"@@"`
}

func (b *opOrTerm) accept(v grammarVisitor) {
	if b.Term != nil {
		b.Term.accept(v)
	}
}

// booleanExpression represents a true/false decision expressed
// as an arbitrary number of terms separated by OR.
type booleanExpression struct {
	Left  *term       `parser:"@@"`
	Right []*opOrTerm `parser:"@@*"`
}

func (b *booleanExpression) checkForCustomError() error {
	validator := &grammarCustomErrorsVisitor{}
	b.accept(validator)
	return validator.join()
}

func (b *booleanExpression) accept(v grammarVisitor) {
	if b.Left != nil {
		b.Left.accept(v)
	}
	for _, r := range b.Right {
		if r != nil {
			r.accept(v)
		}
	}
}

// compareOp is the type of a comparison operator.
type compareOp int

// These are the allowed values of a compareOp
const (
	eq compareOp = iota
	ne
	lt
	lte
	gte
	gt
)

// a fast way to get from a string to a compareOp
var compareOpTable = map[string]compareOp{
	"==": eq,
	"!=": ne,
	"<":  lt,
	"<=": lte,
	">":  gt,
	">=": gte,
}

// Capture is how the parser converts an operator string to a compareOp.
func (c *compareOp) Capture(values []string) error {
	op, ok := compareOpTable[values[0]]
	if !ok {
		return fmt.Errorf("'%s' is not a valid operator", values[0])
	}
	*c = op
	return nil
}

// String() for compareOp gives us more legible test results and error messages.
func (c *compareOp) String() string {
	switch *c {
	case eq:
		return "eq"
	case ne:
		return "ne"
	case lt:
		return "lt"
	case lte:
		return "lte"
	case gte:
		return "gte"
	case gt:
		return "gt"
	default:
		return "UNKNOWN OP!"
	}
}

// comparison represents an optional boolean condition.
type comparison struct {
	Left  value     `parser:"@@"`
	Op    compareOp `parser:"@OpComparison"`
	Right value     `parser:"@@"`
}

func (c *comparison) accept(v grammarVisitor) {
	c.Left.accept(v)
	c.Right.accept(v)
}

// editor represents the function call of a statement.
type editor struct {
	Function  string     `parser:"@(Lowercase(Uppercase | Lowercase)*)"`
	Arguments []argument `parser:"'(' ( @@ ( ',' @@ )* )? ')'"`
	// If keys are matched return an error
	Keys []key `parser:"( @@ )*"`
}

func (i *editor) accept(v grammarVisitor) {
	v.visitEditor(i)
	for _, arg := range i.Arguments {
		arg.accept(v)
	}
}

// converter represents a converter function call.
type converter struct {
	Function  string     `parser:"@(Uppercase(Uppercase | Lowercase)*)"`
	Arguments []argument `parser:"'(' ( @@ ( ',' @@ )* )? ')'"`
	Keys      []key      `parser:"( @@ )*"`
}

func (c *converter) accept(v grammarVisitor) {
	v.visitConverter(c)
	if c.Arguments != nil {
		for _, a := range c.Arguments {
			a.accept(v)
		}
	}
}

type argument struct {
	Name         string  `parser:"(@(Lowercase(Uppercase | Lowercase)*) Equal)?"`
	Value        value   `parser:"( @@"`
	FunctionName *string `parser:"| @(Uppercase(Uppercase | Lowercase)*) )"`
}

func (a *argument) accept(v grammarVisitor) {
	a.Value.accept(v)
}

// value represents a part of a parsed statement which is resolved to a value of some sort. This can be a telemetry path
// mathExpression, function call, or literal.
type value struct {
	IsNil          *isNil           `parser:"( @'nil'"`
	Literal        *mathExprLiteral `parser:"| @@ (?! OpAddSub | OpMultDiv)"`
	MathExpression *mathExpression  `parser:"| @@"`
	Bytes          *byteSlice       `parser:"| @Bytes"`
	String         *string          `parser:"| @String"`
	Bool           *boolean         `parser:"| @Boolean"`
	Enum           *enumSymbol      `parser:"| @Uppercase (?! Lowercase)"`
	Map            *mapValue        `parser:"| @@"`
	List           *list            `parser:"| @@)"`
}

func (v *value) checkForCustomError() error {
	validator := &grammarCustomErrorsVisitor{}
	v.accept(validator)
	return validator.join()
}

func (v *value) accept(vis grammarVisitor) {
	vis.visitValue(v)
	if v.Literal != nil {
		v.Literal.accept(vis)
	}
	if v.MathExpression != nil {
		v.MathExpression.accept(vis)
	}
	if v.Map != nil {
		v.Map.accept(vis)
	}
	if v.List != nil {
		for _, i := range v.List.Values {
			i.accept(vis)
		}
	}
}

// path represents a telemetry path mathExpression.
type path struct {
	Pos     lexer.Position
	Context string  `parser:"(@Lowercase '.')?"`
	Fields  []field `parser:"@@ ( '.' @@ )*"`
}

func (p *path) accept(v grammarVisitor) {
	v.visitPath(p)
	for _, field := range p.Fields {
		field.accept(v)
	}
}

// field is an item within a path.
type field struct {
	Name string `parser:"@Lowercase"`
	Keys []key  `parser:"( @@ )*"`
}

func (f *field) accept(v grammarVisitor) {
	for _, key := range f.Keys {
		key.accept(v)
	}
}

type key struct {
	String         *string          `parser:"'[' (@String "`
	Int            *int64           `parser:"| @Int"`
	MathExpression *mathExpression  `parser:"| @@"`
	Expression     *mathExprLiteral `parser:"| @@ ) ']'"`
}

func (k *key) accept(v grammarVisitor) {
	if k.MathExpression != nil {
		k.MathExpression.accept(v)
	}
	if k.Expression != nil {
		k.Expression.accept(v)
	}
}

type list struct {
	Values []value `parser:"'[' (@@)* (',' @@)* ']'"`
}

type mapValue struct {
	Values []mapItem `parser:"'{' (@@ ','?)* '}'"`
}

func (m *mapValue) accept(v grammarVisitor) {
	for _, i := range m.Values {
		if i.Value != nil {
			i.Value.accept(v)
		}
	}
}

type mapItem struct {
	Key   *string `parser:"@String ':'"`
	Value *value  `parser:"@@"`
}

// byteSlice type for capturing byte slices
type byteSlice []byte

func (b *byteSlice) Capture(values []string) error {
	rawStr := values[0][2:]
	newBytes, err := hex.DecodeString(rawStr)
	if err != nil {
		return err
	}
	*b = newBytes
	return nil
}

// boolean Type for capturing booleans, see:
// https://github.com/alecthomas/participle#capturing-boolean-value
type boolean bool

func (b *boolean) Capture(values []string) error {
	*b = values[0] == "true"
	return nil
}

type isNil bool

func (n *isNil) Capture(_ []string) error {
	*n = true
	return nil
}

type mathExprLiteral struct {
	// If editor is matched then error
	Editor    *editor    `parser:"( @@"`
	Converter *converter `parser:"| @@"`
	Float     *float64   `parser:"| @Float"`
	Int       *int64     `parser:"| @Int"`
	Path      *path      `parser:"| @@ )"`
}

func (m *mathExprLiteral) accept(v grammarVisitor) {
	v.visitMathExprLiteral(m)
	if m.Path != nil {
		m.Path.accept(v)
	}
	if m.Editor != nil {
		m.Editor.accept(v)
	}
	if m.Converter != nil {
		m.Converter.accept(v)
	}
}

type mathValue struct {
	Literal       *mathExprLiteral `parser:"( @@"`
	SubExpression *mathExpression  `parser:"| '(' @@ ')' )"`
}

func (m *mathValue) accept(v grammarVisitor) {
	if m.Literal != nil {
		m.Literal.accept(v)
	}
	if m.SubExpression != nil {
		m.SubExpression.accept(v)
	}
}

type opMultDivValue struct {
	Operator mathOp     `parser:"@OpMultDiv"`
	Value    *mathValue `parser:"@@"`
}

func (m *opMultDivValue) accept(v grammarVisitor) {
	if m.Value != nil {
		m.Value.accept(v)
	}
}

type addSubTerm struct {
	Left  *mathValue        `parser:"@@"`
	Right []*opMultDivValue `parser:"@@*"`
}

func (m *addSubTerm) accept(v grammarVisitor) {
	if m.Left != nil {
		m.Left.accept(v)
	}
	for _, r := range m.Right {
		if r != nil {
			r.accept(v)
		}
	}
}

type opAddSubTerm struct {
	Operator mathOp      `parser:"@OpAddSub"`
	Term     *addSubTerm `parser:"@@"`
}

func (r *opAddSubTerm) accept(v grammarVisitor) {
	if r.Term != nil {
		r.Term.accept(v)
	}
}

type mathExpression struct {
	Left  *addSubTerm     `parser:"@@"`
	Right []*opAddSubTerm `parser:"@@*"`
}

func (m *mathExpression) accept(v grammarVisitor) {
	if m.Left != nil {
		m.Left.accept(v)
	}
	if m.Right != nil {
		for _, r := range m.Right {
			if r != nil {
				r.accept(v)
			}
		}
	}
}

type mathOp int

const (
	add mathOp = iota
	sub
	mult
	div
)

var mathOpTable = map[string]mathOp{
	"+": add,
	"-": sub,
	"*": mult,
	"/": div,
}

func (m *mathOp) Capture(values []string) error {
	op, ok := mathOpTable[values[0]]
	if !ok {
		return fmt.Errorf("'%s' is not a valid operator", values[0])
	}
	*m = op
	return nil
}

func (m *mathOp) String() string {
	switch *m {
	case add:
		return "+"
	case sub:
		return "-"
	case mult:
		return "*"
	case div:
		return "/"
	default:
		return "UNKNOWN OP!"
	}
}

type enumSymbol string

// buildLexer constructs a SimpleLexer definition.
// Note that the ordering of these rules matters.
// It's in a separate function so it can be easily tested alone (see lexer_test.go).
func buildLexer() *lexer.StatefulDefinition {
	return lexer.MustSimple([]lexer.SimpleRule{
		{Name: `Bytes`, Pattern: `0x[a-fA-F0-9]+`},
		{Name: `Float`, Pattern: `[-+]?\d*\.\d+([eE][-+]?\d+)?`},
		{Name: `Int`, Pattern: `[-+]?\d+`},
		{Name: `String`, Pattern: `"(\\.|[^\\"])*"`},
		{Name: `OpNot`, Pattern: `\b(not)\b`},
		{Name: `OpOr`, Pattern: `\b(or)\b`},
		{Name: `OpAnd`, Pattern: `\b(and)\b`},
		{Name: `OpComparison`, Pattern: `==|!=|>=|<=|>|<`},
		{Name: `OpAddSub`, Pattern: `\+|\-`},
		{Name: `OpMultDiv`, Pattern: `\/|\*`},
		{Name: `Boolean`, Pattern: `\b(true|false)\b`},
		{Name: `Equal`, Pattern: `=`},
		{Name: `LParen`, Pattern: `\(`},
		{Name: `RParen`, Pattern: `\)`},
		{Name: `LBrace`, Pattern: `\{`},
		{Name: `RBrace`, Pattern: `\}`},
		{Name: `Colon`, Pattern: `\:`},
		{Name: `Punct`, Pattern: `[,.\[\]]`},
		{Name: `Uppercase`, Pattern: `[A-Z][A-Z0-9_]*`},
		{Name: `Lowercase`, Pattern: `[a-z][a-z0-9_]*`},
		{Name: "whitespace", Pattern: `\s+`},
	})
}

// grammarCustomError represents a grammar error in which the statement has a valid syntax
// according to the grammar's definition, but is still logically invalid.
type grammarCustomError struct {
	errs []error
}

// Error returns all errors messages separate by semicolons.
func (e *grammarCustomError) Error() string {
	switch len(e.errs) {
	case 0:
		return ""
	case 1:
		return e.errs[0].Error()
	default:
		var b strings.Builder
		b.WriteString(e.errs[0].Error())
		for _, err := range e.errs[1:] {
			b.WriteString("; ")
			b.WriteString(err.Error())
		}
		return b.String()
	}
}

func (e *grammarCustomError) Unwrap() []error {
	return e.errs
}

// grammarVisitor allows accessing the grammar AST nodes using the visitor pattern.
type grammarVisitor interface {
	visitPath(v *path)
	visitEditor(v *editor)
	visitConverter(v *converter)
	visitValue(v *value)
	visitMathExprLiteral(v *mathExprLiteral)
}

// grammarCustomErrorsVisitor is used to execute custom validations on the grammar AST.
type grammarCustomErrorsVisitor struct {
	errs []error
}

func (g *grammarCustomErrorsVisitor) add(err error) {
	g.errs = append(g.errs, err)
}

func (g *grammarCustomErrorsVisitor) join() error {
	if len(g.errs) == 0 {
		return nil
	}
	return &grammarCustomError{errs: g.errs}
}

func (g *grammarCustomErrorsVisitor) visitPath(_ *path) {}

func (g *grammarCustomErrorsVisitor) visitValue(_ *value) {}

func (g *grammarCustomErrorsVisitor) visitConverter(_ *converter) {}

func (g *grammarCustomErrorsVisitor) visitEditor(v *editor) {
	if v.Keys != nil {
		g.add(fmt.Errorf("only paths and converters may be indexed, not editors, but got %s%s", v.Function, buildOriginalKeysText(v.Keys)))
	}
}

func (g *grammarCustomErrorsVisitor) visitMathExprLiteral(v *mathExprLiteral) {
	if v.Editor != nil {
		g.add(fmt.Errorf("converter names must start with an uppercase letter but got '%v'", v.Editor.Function))
	}
}
