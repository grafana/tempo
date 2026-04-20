package traceql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalQuery(t *testing.T) {
	// Expected values go through the AST stringer which wraps non-leaf nodes
	// in parentheses (see wrapElement in ast_stringer.go), so conditions like
	// `.foo = "bar"` become `(.foo = "bar")` when part of a larger expression.
	// This doesn't change the semantic meaning of the query.
	testCases := []struct {
		name, query, expected string
	}{
		{
			name:     "empty query",
			query:    "",
			expected: "{}",
		},
		{
			name:     "empty query with spaces",
			query:    " { } ",
			expected: "{}",
		},
		{
			name:     "simple query",
			query:    `{.service_name = "foo"}`,
			expected: `{.service_name = "foo"}`,
		},
		{
			name:     "incomplete query",
			query:    `{ .http.status_code = 200 && .http.method = }`,
			expected: "{(.http.status_code = 200) && .http.method}",
		},
		{
			name:     "reversed operands with missing closing bracket",
			query:    "{ 2 = .b ",
			expected: "{ 2 = .b}",
		},
		{
			name:     "long query",
			query:    `{.service_name = "foo" && .http.status_code = 200 && .http.method = "GET" && .cluster = }`,
			expected: `{(((.service_name = "foo") && (.http.status_code = 200)) && (.http.method = "GET")) && .cluster}`,
		},
		{
			name:     "query with duration a boolean",
			query:    `{ duration > 5s && .success = true && .cluster = }`,
			expected: `{((duration > 5s) && (.success = true)) && .cluster}`,
		},
		{
			name:     "query with three selectors with AND",
			query:    `{ .foo = "bar" && .baz = "qux" } && { duration > 1s } || { .foo = "bar" && .baz = "qux" }`,
			expected: `(({ (.foo = "bar") && (.baz = "qux") }) && ({ duration > 1s })) || ({ (.foo = "bar") && (.baz = "qux") })`,
		},
		{
			name:     "query with OR conditions",
			query:    `{ (.foo = "bar" || .baz = "qux") && duration > 1s }`,
			expected: `{((.foo = "bar") || (.baz = "qux")) && (duration > 1s)}`,
		},
		{
			name:     "query with multiple selectors and pipelines",
			query:    `{ .foo = "bar" && .baz = "qux" } && { duration > 1s } || { .foo = "bar" && .baz = "qux" } | count() > 4`,
			expected: `(({ (.foo = "bar") && (.baz = "qux") }) && ({ duration > 1s })) || ({ (.foo = "bar") && (.baz = "qux") }) | (count()) > 4`,
		},
		{
			name:     "query with slash in value",
			query:    `{ span.http.target = "/api/v1/users" }`,
			expected: `{span.http.target = "/api/v1/users"}`,
		},
		{
			name:     "intrinsics",
			query:    `{  name = "foo" }`,
			expected: `{name = "foo"}`,
		},
		{
			name:     "incomplete intrinsics",
			query:    `{  statusMessage = }`,
			expected: "{}",
		},
		{
			name:     "query with missing closing bracket",
			query:    `{resource.service_name = "foo" && span.http.target=`,
			expected: `{(resource.service_name = "foo") && span.http.target}`,
		},
		{
			name:     "uncommon characters",
			query:    `{ span.foo = "<>:b5[]" && resource.service.name = }`,
			expected: `{(span.foo = "<>:b5[]") && resource.service.name}`,
		},
		{
			name:     "kind",
			query:    `{ kind = server }`,
			expected: `{kind = server}`,
		},
		{
			name:     "attribute with dashes",
			query:    `{ span.foo-bar = "baz" }`,
			expected: `{span.foo-bar = "baz"}`,
		},
		{
			name:     "attribute with quotes and spaces",
			query:    `{ span."foo bar" = "baz" }`,
			expected: `{span."foo bar" = "baz"}`,
		},
		{
			name:     "query with trivial regex matcher",
			query:    `{ .foo =~ "a" }`,
			expected: `{.foo =~ "a"}`,
		},
		{
			name:     "query with regex matcher",
			query:    `{ .foo =~ "(a|b)" }`,
			expected: `{.foo =~ "(a|b)"}`,
		},
		{
			name:     "query with multiple regex matchers",
			query:    `{ .foo =~ "(a|b)" && .bar =~ "(c|d)" }`,
			expected: `{(.foo =~ "(a|b)") && (.bar =~ "(c|d)")}`,
		},
		{
			name:     "query with mixed equal and regex matchers",
			query:    `{ .foo = "a" && .bar =~ "(c|d)" }`,
			expected: `{(.foo = "a") && (.bar =~ "(c|d)")}`,
		},
		{
			name:     "scoped intrinsic",
			query:    `{ event:name = "exception" }`,
			expected: `{event:name = "exception"}`,
		},
		{
			name:     "whitespace in value",
			query:    `{ .foo = " b a r " }   `,
			expected: `{.foo = " b a r "}`,
		},
		{
			name:     "structural operators with incomplete in first matcher",
			query:    `{ .foo = "bar" && .baaz = } >> { .bar = "foo" }`,
			expected: `({ (.foo = "bar") && .baaz }) >> ({ .bar = "foo" })`,
		},
		{
			name:     "structural operators with incomplete in second matcher",
			query:    `{ .foo = "bar" } >> { .bar = }`,
			expected: `({ .foo = "bar" }) >> ({ true })`,
		},
		{
			name:     "metrics query",
			query:    `{.service_name = "foo" && .foo=} | rate() by (.bar)`,
			expected: `{(.service_name = "foo") && .foo} | rate() by (.bar)`,
		},
		{
			name:     "query with select",
			query:    `{.service_name = "foo" && .foo=} | select(.bar, .baz)`,
			expected: `{(.service_name = "foo") && .foo} | select(.bar, .baz)`,
		},
		{
			name:     "query with parentheses and incomplete matcher",
			query:    `{ (resource.foo = "bar" && .baz = ) && .qux = "quux" }`,
			expected: `{((resource.foo = "bar") && .baz) && (.qux = "quux")}`,
		},
		{
			name:     "query with parentheses containing all incomplete matchers",
			query:    `{ (resource.foo =  && .baz = ) && .qux = "quux" }`,
			expected: `{(resource.foo && .baz) && (.qux = "quux")}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expected := tc.expected
			expected = strings.ReplaceAll(expected, " ", "")
			actual := NormalizeQuery(tc.query)
			actual = strings.ReplaceAll(actual, " ", "")
			actual = strings.ReplaceAll(actual, "`", `"`)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestExtractConditionGroups(t *testing.T) {
	testCases := []struct {
		name  string
		query string
		count int
	}{
		{name: "empty", query: "", count: 0},
		{name: "empty braces", query: " { } ", count: 0},
		{name: "simple", query: `{.service_name = "foo"}`, count: 1},
		{name: "incomplete", query: `{ .http.status_code = 200 && .http.method = }`, count: 1},
		{name: "invalid", query: "{ invalid syntax }", count: 0},
		{name: "OR conditions", query: `{ (.foo = "bar" || .baz = "qux") }`, count: 2},
		{name: "structural", query: `{ .foo = "bar" } >> { .bar = "baz" }`, count: 0},
		{name: "multiple conditions", query: `{.a = 1 && .b = "two" && .c > 3}`, count: 1},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conditionGroups, _ := ExtractConditionGroups(tc.query, DefaultMaxConditionGroupsPerTagQuery)
			assert.Equal(t, tc.count, len(conditionGroups))
		})
	}
}

func BenchmarkCanonicalQuery(b *testing.B) {
	queries := []string{
		`{.service_name = "foo"}`,
		`{.service_name = "foo" && .http.status_code = 200}`,
		`{.service_name = "foo" && .http.status_code = 200 && .http.method = "GET"}`,
		`{.service_name = "foo" && .http.status_code = 200 && .http.method = "GET" && .http.url = "/foo"}`,
		`{.service_name = "foo" && .cluster = }`,
		`{.service_name = "foo" && .http.status_code = 200 && .cluster = }`,
		`{.service_name = "foo" && .http.status_code = 200 && .http.method = "GET" && .cluster = }`,
		`{.service_name = "foo" && .http.status_code = 200 && .http.method = "GET" && .http.url = "/foo" && .cluster = }`,
	}
	for _, query := range queries {
		b.Run(query, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = NormalizeQuery(query)
			}
		})
	}
}

func TestFlattenExprToOperations(t *testing.T) {
	type testOperation struct {
		opType Operator
		cond   []int
	}

	testCases := []struct {
		name, query string
		expected    []testOperation
	}{
		{
			name:     "three single condition ORs",
			query:    `{ .attr = "123" || .service = "b" || .env = "staging" }`,
			expected: []testOperation{{opType: OpOr, cond: []int{1, 1, 1}}},
		},
		{
			name:     "one double condition ORs with two single condition ORs",
			query:    `{ (.attr = "123" && .foo = "bar") || .service = "b" || .env = "staging" }`,
			expected: []testOperation{{opType: OpOr, cond: []int{2, 1, 1}}},
		},
		{
			name:     "two duplicated OR conditions",
			query:    `{ (.attr = "123" && .foo = "bar") || .service = "b" || .env = "staging" || (.attr = "123" && .foo = "bar") || .service = "b" }`,
			expected: []testOperation{{opType: OpOr, cond: []int{2, 1, 1}}},
		},
		{
			name:     "one single condition AND three single condition ORs",
			query:    `{ name = "abc" && (.attr = "123" || .service = "b" || .env = "staging") }`,
			expected: []testOperation{{opType: OpAnd, cond: []int{1}}, {opType: OpOr, cond: []int{1, 1, 1}}},
		},
		{
			name:     "one AND one double condition OR with two single condition ORs",
			query:    `{ name = "abc" && ( (.attr = "123" && .foo = "bar") || .service = "b" || .env = "staging") }`,
			expected: []testOperation{{opType: OpAnd, cond: []int{1}}, {opType: OpOr, cond: []int{2, 1, 1}}},
		},
		{
			name:     "two ANDs one double condition OR with two single condition ORs",
			query:    `{ name = "abc" && .attr = "abc" && ( (.attr = "123" && .foo = "bar") || .service = "b" || .env = "staging") }`,
			expected: []testOperation{{opType: OpAnd, cond: []int{1}}, {opType: OpAnd, cond: []int{1}}, {opType: OpOr, cond: []int{2, 1, 1}}},
		},
		{
			name:     "two AND with single condition OR in between",
			query:    `{ .attr = "123" && (.service = "b" || .service = "a" ) && .env = "staging" }`,
			expected: []testOperation{{opType: OpAnd, cond: []int{1}}, {opType: OpOr, cond: []int{1, 1}}, {opType: OpAnd, cond: []int{1}}},
		},
		{
			name:     "two ANDs of single condition ORs",
			query:    `{ ( .attr = "123" || .service = "b" ) && ( .service = "a" || .env = "staging" ) }`,
			expected: []testOperation{{opType: OpOr, cond: []int{1, 1}}, {opType: OpOr, cond: []int{1, 1}}},
		},
		{
			name:     "two ANDs of multiple conditions ORs",
			query:    `{ ( .attr = "123" || .service = "b" ) && ( .service = "a" || ( .env = "staging" && .foo = "bar" ) ) }`,
			expected: []testOperation{{opType: OpOr, cond: []int{1, 1}}, {opType: OpOr, cond: []int{1, 2}}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := ParseNoOptimizations(tc.query)
			assert.NoError(t, err)

			operations := flattenExprToOperations(expr.Pipeline.Elements[0].(*SpansetFilter).Expression, OpNone)
			assert.NoError(t, err)
			assert.Equal(t, len(tc.expected), len(operations), "expected %d operations, got %d", len(tc.expected), len(operations))
			for i, operationCount := range tc.expected {
				assert.Equal(t, operationCount.opType, operations[i].opType, "expected operation type %v at index %d, got %v", operationCount.opType, i, operations[i].opType)
				assert.Equal(t, len(operationCount.cond), len(operations[i].conditions), "expected %d conditions at index %d, got %d", len(operationCount.cond), i, len(operations[i].conditions))
				for j, conditionCount := range operationCount.cond {
					assert.Equal(t, conditionCount, len(operations[i].conditions[j]), "expected %d conditions at index %d.%d, got %d", conditionCount, i, j, len(operations[i].conditions[j]))
				}
			}
		})
	}
}

func TestSplitReqConditionGroups(t *testing.T) {
	testCases := []struct {
		name, query string
		expected    [][]Condition
	}{
		{
			name:  "simple no OR query",
			query: `{ .attr = "123" }`,
			expected: [][]Condition{
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
				},
			},
		},
		{
			name:     "opnone query",
			query:    `{.attr}`,
			expected: nil,
		},
		{
			name:     "one OR is opnone query",
			query:    `{.attr || .foo = "bar" }`,
			expected: nil,
		},
		{
			name:  "three single condition ORs",
			query: `{ .attr = "123" || .service = "b" || .env = "staging" }`,
			expected: [][]Condition{
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
				},
				{
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
				},
				{
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
			},
		},
		{
			name:  "one double condition ORs with two single condition ORs",
			query: `{ (.attr = "123" && .foo = "bar") || .service = "b" || .env = "staging" }`,
			expected: [][]Condition{
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				},
				{
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
				},
				{
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
			},
		},
		{
			name:  "one single condition AND three single condition ORs",
			query: `{ name = "abc" && (.attr = "123" || .service = "b" || .env = "staging") }`,
			expected: [][]Condition{
				{
					newCondition(NewIntrinsic(IntrinsicName), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
				},
				{
					newCondition(NewIntrinsic(IntrinsicName), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
				},
				{
					newCondition(NewIntrinsic(IntrinsicName), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
			},
		},
		{
			name:  "one AND one double condition OR with two single condition ORs",
			query: `{ name = "abc" && ( (.attr = "123" && .foo = "bar") || .service = "b" || .env = "staging") }`,
			expected: [][]Condition{
				{
					newCondition(NewIntrinsic(IntrinsicName), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				},
				{
					newCondition(NewIntrinsic(IntrinsicName), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
				},
				{
					newCondition(NewIntrinsic(IntrinsicName), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
			},
		},
		{
			name:  "two ANDs one double condition OR with two single condition ORs",
			query: `{ name = "abc" && .attr = "abc" && ( (.attr = "123" && .foo = "bar") || .service = "b" || .env = "staging") }`,
			expected: [][]Condition{
				{
					newCondition(NewIntrinsic(IntrinsicName), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				},
				{
					newCondition(NewIntrinsic(IntrinsicName), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
				},
				{
					newCondition(NewIntrinsic(IntrinsicName), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("abc")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
			},
		},
		{
			name:  "two AND with single condition OR in between",
			query: `{ .attr = "123" && (.service = "b" || .service = "a" ) && .env = "staging" }`,
			expected: [][]Condition{
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("a")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
			},
		},
		{
			name:  "two ANDs of single condition ORs",
			query: `{ ( .attr = "123" || .service = "b" ) && ( .service = "a" || .env = "staging"  || .foo = "bar" ) }`,
			expected: [][]Condition{
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("a")),
				},
				{
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("a")),
				},
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
				{
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				},
				{
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				},
			},
		},
		{
			name:  "two ANDs of multiple conditions ORs",
			query: `{ ( .attr = "123" || .service = "b" ) && ( .service = "a" || ( .env = "staging" && .foo = "bar" ) ) }`,
			expected: [][]Condition{
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("a")),
				},
				{
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("a")),
				},
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				},
				{
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				},
			},
		},
		{
			name:  "three ANDs of two single condition ORs",
			query: `{ ( .attr = "123" || .service = "b" ) && .team = "dev" && ( .service = "a" || .env = "staging"  || .foo = "bar" ) }`,
			expected: [][]Condition{
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("team"), OpEqual, NewStaticString("dev")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("a")),
				},
				{
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
					newCondition(NewAttribute("team"), OpEqual, NewStaticString("dev")),
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("a")),
				},
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("team"), OpEqual, NewStaticString("dev")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
				{
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
					newCondition(NewAttribute("team"), OpEqual, NewStaticString("dev")),
					newCondition(NewAttribute("env"), OpEqual, NewStaticString("staging")),
				},
				{
					newCondition(NewAttribute("attr"), OpEqual, NewStaticString("123")),
					newCondition(NewAttribute("team"), OpEqual, NewStaticString("dev")),
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				},
				{
					newCondition(NewAttribute("service"), OpEqual, NewStaticString("b")),
					newCondition(NewAttribute("team"), OpEqual, NewStaticString("dev")),
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("bar")),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conditions, err := ExtractConditionGroups(tc.query, DefaultMaxConditionGroupsPerTagQuery)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, conditions)
		})
	}
}

func TestSplitReqConditionGroupsMaxGroups(t *testing.T) {
	longQuery := `{ .attr = "123" || .service = "b" || .team = "dev" || .service = "a" || .env = "staging"  || .foo = "bar" || .bar = "foo" || .baz = "qux" || .quux = "corge" || .grault = "garply" || .waldo = "fred" || .plugh = "xyzzy" || .thud = "mno" }`
	testCases := []struct {
		name           string
		query          string
		maxGroups      int
		expectedLength int
		expectError    bool
	}{
		{
			name:           "max groups reached - limit of 10",
			query:          longQuery,
			maxGroups:      10,
			expectedLength: 0,
			expectError:    true,
		},
		{
			name:           "max groups reached - limit of 5",
			query:          longQuery,
			maxGroups:      5,
			expectedLength: 0,
			expectError:    true,
		},
		{
			name:           "within limit - default limit",
			query:          `{ .a = "1" || .b = "2" || .c = "3" }`,
			maxGroups:      DefaultMaxConditionGroupsPerTagQuery,
			expectedLength: 3,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conditions, err := ExtractConditionGroups(tc.query, tc.maxGroups)
			if tc.expectError {
				assert.ErrorIs(t, err, ErrMaxConditionGroupsPerTagQueryReached)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedLength, len(conditions))
		})
	}
}
