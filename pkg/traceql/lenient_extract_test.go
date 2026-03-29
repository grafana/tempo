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
			conditionGroups := ExtractConditionGroups(tc.query)
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
