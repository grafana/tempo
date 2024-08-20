package traceql

import (
	"strings"
	"testing"
	"text/scanner"
	"time"

	"github.com/stretchr/testify/require"
)

type lexerTestCase struct {
	input    string
	expected []int
}

func TestLexerAttributes(t *testing.T) {
	testLexer(t, ([]lexerTestCase{
		// attributes
		{`.foo`, []int{DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`."foo".baz."bar"`, []int{DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.foo."bar \" baz"."bar"`, []int{DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.foo."baz \\".bar`, []int{DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`."foo.bar"`, []int{DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.count`, []int{DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.foo3`, []int{DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.foo+bar`, []int{DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.foo-bar`, []int{DOT, IDENTIFIER, END_ATTRIBUTE}},
		// parent attributes
		{`parent.foo`, []int{PARENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.count`, []int{PARENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.foo3`, []int{PARENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.foo+bar`, []int{PARENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.foo-bar`, []int{PARENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		// span attributes
		{`span.foo`, []int{SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`span.count`, []int{SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`span.foo3`, []int{SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`span.foo+bar`, []int{SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`span.foo-bar`, []int{SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		// resource attributes
		{`resource.foo`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`resource.count`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`resource.foo3`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`resource.foo+bar`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`resource.foo-bar`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		// event attributes
		{`event.foo`, []int{EVENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`event.count`, []int{EVENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`event.count`, []int{EVENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`event.foo3`, []int{EVENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`event.foo+bar`, []int{EVENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`event.foo-bar`, []int{EVENT_DOT, IDENTIFIER, END_ATTRIBUTE}},
		// link attributes
		{`link.foo`, []int{LINK_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`link.count`, []int{LINK_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`link.count`, []int{LINK_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`link.foo3`, []int{LINK_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`link.foo+bar`, []int{LINK_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`link.foo-bar`, []int{LINK_DOT, IDENTIFIER, END_ATTRIBUTE}},
		// instrumentation attributes
		{`instrumentation.foo`, []int{INSTRUMENTATION_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`instrumentation.count`, []int{INSTRUMENTATION_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`instrumentation.foo3`, []int{INSTRUMENTATION_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`instrumentation.foo+bar`, []int{INSTRUMENTATION_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`instrumentation.foo-bar`, []int{INSTRUMENTATION_DOT, IDENTIFIER, END_ATTRIBUTE}},
		// parent span attributes
		{`parent.span.foo`, []int{PARENT_DOT, SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.span.count`, []int{PARENT_DOT, SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.span.foo3`, []int{PARENT_DOT, SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.span.foo+bar`, []int{PARENT_DOT, SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.span.foo-bar`, []int{PARENT_DOT, SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		// parent resource attributes
		{`parent.resource.foo`, []int{PARENT_DOT, RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.resource.count`, []int{PARENT_DOT, RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.resource.foo3`, []int{PARENT_DOT, RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.resource.foo+bar`, []int{PARENT_DOT, RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.resource.foo-bar`, []int{PARENT_DOT, RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		// attribute enders: <space>, {, }, (, ), <comma> all force end an attribute
		{`.foo .bar`, []int{DOT, IDENTIFIER, END_ATTRIBUTE, DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.foo}.bar`, []int{DOT, IDENTIFIER, END_ATTRIBUTE, CLOSE_BRACE, DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.foo{.bar`, []int{DOT, IDENTIFIER, END_ATTRIBUTE, OPEN_BRACE, DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.foo).bar`, []int{DOT, IDENTIFIER, END_ATTRIBUTE, CLOSE_PARENS, DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.foo(.bar`, []int{DOT, IDENTIFIER, END_ATTRIBUTE, OPEN_PARENS, DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`.foo,.bar`, []int{DOT, IDENTIFIER, END_ATTRIBUTE, COMMA, DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`. foo`, []int{DOT, END_ATTRIBUTE, IDENTIFIER}},
		// not attributes
		{`.3`, []int{FLOAT}},
		{`.24h`, []int{DURATION}},
	}))
}

func TestLexerScopedIntrinsic(t *testing.T) {
	testLexer(t, ([]lexerTestCase{
		// trace scoped intrinsics
		{`trace:duration`, []int{TRACE_COLON, IDURATION}},
		{`trace:rootName`, []int{TRACE_COLON, ROOTNAME}},
		{`trace:rootService`, []int{TRACE_COLON, ROOTSERVICE}},
		{`trace:id`, []int{TRACE_COLON, ID}},
		// span scoped intrinsics
		{`span:duration`, []int{SPAN_COLON, IDURATION}},
		{`span:name`, []int{SPAN_COLON, NAME}},
		{`span:kind`, []int{SPAN_COLON, KIND}},
		{`span:status`, []int{SPAN_COLON, STATUS}},
		{`span:statusMessage`, []int{SPAN_COLON, STATUS_MESSAGE}},
		{`span:id`, []int{SPAN_COLON, ID}},
		// event scoped intrinsics
		{`event:name`, []int{EVENT_COLON, NAME}},
		{`event:timeSinceStart`, []int{EVENT_COLON, TIMESINCESTART}},
		// link scoped intrinsics
		{`link:traceID`, []int{LINK_COLON, TRACE_ID}},
		{`link:spanID`, []int{LINK_COLON, SPAN_ID}},
		// instrumentation scoped intrinsics
		{`instrumentation:name`, []int{INSTRUMENTATION_COLON, NAME}},
		{`instrumentation:version`, []int{INSTRUMENTATION_COLON, VERSION}},
	}))
}

func TestLexerIntrinsics(t *testing.T) {
	testLexer(t, ([]lexerTestCase{
		{`nestedSetLeft`, []int{NESTEDSETLEFT}},
		{`nestedSetRight`, []int{NESTEDSETRIGHT}},
		{`duration`, []int{IDURATION}},
	}))
}

func TestLexerMultitokens(t *testing.T) {
	testLexer(t, ([]lexerTestCase{
		// attributes
		{`&&`, []int{AND}},
		{`>>`, []int{DESC}},
		{`!<<`, []int{NOT_ANCE}},
		{`!`, []int{NOT}},
		{`!~`, []int{NRE}},
		{`&>>`, []int{UNION_DESC}},
	}))
}

func TestLexerDuration(t *testing.T) {
	testLexer(t, ([]lexerTestCase{
		// duration
		{"1ns", []int{DURATION}},
		{"1s", []int{DURATION}},
		{"1us", []int{DURATION}},
		{"1m", []int{DURATION}},
		{"1h", []int{DURATION}},
		{"1µs", []int{DURATION}},
		{"1y", []int{DURATION}},
		{"1w", []int{DURATION}},
		{"1d", []int{DURATION}},
		{"1h15m30.918273645s", []int{DURATION}},
		// not duration
		{"1t", []int{INTEGER, IDENTIFIER}},
		{"1", []int{INTEGER}},
	}))
}

func TestLexerParseDuration(t *testing.T) {
	const MICROSECOND = 1000 * time.Nanosecond
	const DAY = 24 * time.Hour
	const WEEK = 7 * DAY
	const YEAR = 365 * DAY

	for _, tc := range []struct {
		input    string
		expected time.Duration
	}{
		{"1ns", time.Nanosecond},
		{"1s", time.Second},
		{"1us", MICROSECOND},
		{"1m", time.Minute},
		{"1h", time.Hour},
		{"1µs", MICROSECOND},
		{"1y", YEAR},
		{"1w", WEEK},
		{"1d", DAY},
		{"1h15m30.918273645s", time.Hour + 15*time.Minute + 30*time.Second + 918273645*time.Nanosecond},
	} {
		actual, err := parseDuration(tc.input)

		require.Equal(t, err, nil)
		require.Equal(t, tc.expected, actual)
	}
}

func TestLexerScoping(t *testing.T) {
	testLexer(t, ([]lexerTestCase{
		{`span.foo3`, []int{SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`span.foo+bar`, []int{SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`span.foo-bar`, []int{SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`span.resource.foo`, []int{SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.span.foo`, []int{PARENT_DOT, SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`span.foo.bar`, []int{SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		// resource attributes
		{`resource.foo`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`resource.count`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`resource.foo3`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`resource.foo+bar`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`resource.foo-bar`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`resource.span.foo`, []int{RESOURCE_DOT, IDENTIFIER, END_ATTRIBUTE}},
		// parent span attributes
		{`parent.span.foo`, []int{PARENT_DOT, SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.span.count`, []int{PARENT_DOT, SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
		{`parent.span.resource.id`, []int{PARENT_DOT, SPAN_DOT, IDENTIFIER, END_ATTRIBUTE}},
	}))
}

func testLexer(t *testing.T, tcs []lexerTestCase) {
	for _, tc := range tcs {
		t.Run(tc.input, func(t *testing.T) {
			actual := []int{}
			l := lexer{
				Scanner: scanner.Scanner{
					Mode: scanner.SkipComments | scanner.ScanStrings,
				},
			}
			l.Init(strings.NewReader(tc.input))
			var lval yySymType
			for {
				tok := l.Lex(&lval)
				if tok == 0 {
					break
				}
				actual = append(actual, tok)
			}

			require.Equal(t, tc.expected, actual)
		})
	}
}

func BenchmarkIsAttributeRune(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isAttributeRune('=')
	}
}
