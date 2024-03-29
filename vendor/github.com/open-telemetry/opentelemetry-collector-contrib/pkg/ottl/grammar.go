// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"encoding/hex"
	"fmt"

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
	if p.Converter != nil {
		return fmt.Errorf("editor names must start with a lowercase letter but got '%v'", p.Converter.Function)
	}
	err := p.Editor.checkForCustomError()
	if err != nil {
		return err
	}
	if p.WhereClause != nil {
		return p.WhereClause.checkForCustomError()
	}
	return nil
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

func (b *booleanValue) checkForCustomError() error {
	if b.Comparison != nil {
		return b.Comparison.checkForCustomError()
	}
	if b.SubExpr != nil {
		return b.SubExpr.checkForCustomError()
	}
	return nil
}

// opAndBooleanValue represents the right side of an AND boolean expression.
type opAndBooleanValue struct {
	Operator string        `parser:"@OpAnd"`
	Value    *booleanValue `parser:"@@"`
}

func (b *opAndBooleanValue) checkForCustomError() error {
	return b.Value.checkForCustomError()
}

// term represents an arbitrary number of boolean values joined by AND.
type term struct {
	Left  *booleanValue        `parser:"@@"`
	Right []*opAndBooleanValue `parser:"@@*"`
}

func (b *term) checkForCustomError() error {
	err := b.Left.checkForCustomError()
	if err != nil {
		return err
	}
	for _, r := range b.Right {
		err = r.checkForCustomError()
		if err != nil {
			return err
		}
	}
	return nil
}

// opOrTerm represents the right side of an OR boolean expression.
type opOrTerm struct {
	Operator string `parser:"@OpOr"`
	Term     *term  `parser:"@@"`
}

func (b *opOrTerm) checkForCustomError() error {
	return b.Term.checkForCustomError()
}

// booleanExpression represents a true/false decision expressed
// as an arbitrary number of terms separated by OR.
type booleanExpression struct {
	Left  *term       `parser:"@@"`
	Right []*opOrTerm `parser:"@@*"`
}

func (b *booleanExpression) checkForCustomError() error {
	err := b.Left.checkForCustomError()
	if err != nil {
		return err
	}
	for _, r := range b.Right {
		err = r.checkForCustomError()
		if err != nil {
			return err
		}
	}
	return nil
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

func (c *comparison) checkForCustomError() error {
	err := c.Left.checkForCustomError()
	if err != nil {
		return err
	}
	err = c.Right.checkForCustomError()
	return err
}

// editor represents the function call of a statement.
type editor struct {
	Function  string     `parser:"@(Lowercase(Uppercase | Lowercase)*)"`
	Arguments []argument `parser:"'(' ( @@ ( ',' @@ )* )? ')'"`
	// If keys are matched return an error
	Keys []key `parser:"( @@ )*"`
}

func (i *editor) checkForCustomError() error {
	var err error

	for _, arg := range i.Arguments {
		err = arg.checkForCustomError()
		if err != nil {
			return err
		}
	}
	if i.Keys != nil {
		return fmt.Errorf("only paths and converters may be indexed, not editors, but got %v %v", i.Function, i.Keys)
	}
	return nil
}

// converter represents a converter function call.
type converter struct {
	Function  string     `parser:"@(Uppercase(Uppercase | Lowercase)*)"`
	Arguments []argument `parser:"'(' ( @@ ( ',' @@ )* )? ')'"`
	Keys      []key      `parser:"( @@ )*"`
}

type argument struct {
	Name  string `parser:"(@(Lowercase(Uppercase | Lowercase)*) Equal)?"`
	Value value  `parser:"@@"`
}

func (a *argument) checkForCustomError() error {
	return a.Value.checkForCustomError()
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
	FunctionName   *string          `parser:"| @(Uppercase(Uppercase | Lowercase)*)"`
	List           *list            `parser:"| @@)"`
}

func (v *value) checkForCustomError() error {
	if v.Literal != nil {
		return v.Literal.checkForCustomError()
	}
	if v.MathExpression != nil {
		return v.MathExpression.checkForCustomError()
	}
	return nil
}

// path represents a telemetry path mathExpression.
type path struct {
	Fields []field `parser:"@@ ( '.' @@ )*"`
}

// field is an item within a path.
type field struct {
	Name string `parser:"@Lowercase"`
	Keys []key  `parser:"( @@ )*"`
}

type key struct {
	String *string `parser:"'[' (@String "`
	Int    *int64  `parser:"| @Int) ']'"`
}

type list struct {
	Values []value `parser:"'[' (@@)* (',' @@)* ']'"`
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

func (m *mathExprLiteral) checkForCustomError() error {
	if m.Editor != nil {
		return fmt.Errorf("converter names must start with an uppercase letter but got '%v'", m.Editor.Function)
	}
	return nil
}

type mathValue struct {
	Literal       *mathExprLiteral `parser:"( @@"`
	SubExpression *mathExpression  `parser:"| '(' @@ ')' )"`
}

func (m *mathValue) checkForCustomError() error {
	if m.Literal != nil {
		return m.Literal.checkForCustomError()
	}
	return m.SubExpression.checkForCustomError()
}

type opMultDivValue struct {
	Operator mathOp     `parser:"@OpMultDiv"`
	Value    *mathValue `parser:"@@"`
}

func (m *opMultDivValue) checkForCustomError() error {
	return m.Value.checkForCustomError()
}

type addSubTerm struct {
	Left  *mathValue        `parser:"@@"`
	Right []*opMultDivValue `parser:"@@*"`
}

func (m *addSubTerm) checkForCustomError() error {
	err := m.Left.checkForCustomError()
	if err != nil {
		return err
	}
	for _, r := range m.Right {
		err = r.checkForCustomError()
		if err != nil {
			return err
		}
	}
	return nil
}

type opAddSubTerm struct {
	Operator mathOp      `parser:"@OpAddSub"`
	Term     *addSubTerm `parser:"@@"`
}

func (m *opAddSubTerm) checkForCustomError() error {
	return m.Term.checkForCustomError()
}

type mathExpression struct {
	Left  *addSubTerm     `parser:"@@"`
	Right []*opAddSubTerm `parser:"@@*"`
}

func (m *mathExpression) checkForCustomError() error {
	err := m.Left.checkForCustomError()
	if err != nil {
		return err
	}
	for _, r := range m.Right {
		err = r.checkForCustomError()
		if err != nil {
			return err
		}
	}
	return nil
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
		{Name: `Punct`, Pattern: `[,.\[\]]`},
		{Name: `Uppercase`, Pattern: `[A-Z][A-Z0-9_]*`},
		{Name: `Lowercase`, Pattern: `[a-z][a-z0-9_]*`},
		{Name: "whitespace", Pattern: `\s+`},
	})
}
