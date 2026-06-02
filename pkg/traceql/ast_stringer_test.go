package traceql

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

const testExamplesFile = "./test_examples.yaml"

func TestStringer(t *testing.T) {
	b, err := os.ReadFile(testExamplesFile)
	require.NoError(t, err)

	queries := &TestQueries{}
	err = yaml.Unmarshal(b, queries)
	require.NoError(t, err)

	// All of these queries are parseable and valid constructs, and should be roundtrippable.
	sets := [][]string{
		queries.Valid,
		queries.Unsupported,
	}

	for _, s := range sets {
		for _, q := range s {
			t.Run(q, func(t *testing.T) {
				pass1, err := ParseNoOptimizations(q)
				require.NoError(t, err)

				// now parse it a second time and confirm that it parses the same way twice
				pass2, err := ParseNoOptimizations(pass1.String())
				ok := assert.NoError(t, err)
				if !ok {
					t.Logf("\n\t1: %s", pass1.String())
					return
				}

				assert.Equal(t, pass1, pass2)
				t.Logf("\n\tq: %s\n\t1: %s\n\t2: %s", q, pass1.String(), pass2.String())
			})
		}
	}
}

func TestStringerRoundtrip(t *testing.T) {
	roundtrippable := []string{
		"{ duration = 1s }", // Check handling of some tricky parts where intrinsics / attributes overlap
		"{ .duration = 1s }",
		"{ parent.duration = 1s }",
		"{ span.duration = 1s }",
		"{ resource.duration = 1s }",
	}

	for _, q := range roundtrippable {
		t.Run(q, func(t *testing.T) {
			expr, err := Parse(q)
			require.NoError(t, err)
			require.Equal(t, q, expr.String())
		})
	}
}

func TestStringerAttributeNameWithQuote(t *testing.T) {
	queries := []string{
		// " and \ escapes
		`{ ."foo\" = \"bar" = "v" }`,
		`{ ."a\\b" = "v" }`,
		`{ ."a\"b c" = "v" }`,
		// name contains only attribute runes plus a literal '"' — the stringer
		// must still force-quote because '"' is an attribute rune per
		// isAttributeRune, so ContainsNonAttributeRune would otherwise miss it.
		`{ ."a\"b" = "v" }`,
		// raw whitespace / control / non-printable runes — the parser accepts
		// them verbatim inside a quoted identifier; a strconv.Quote-based
		// serializer would emit \n / \t / \uXXXX and break re-parse.
		"{ .\"foo\tbar\" = \"v\" }",
		"{ .\"foo\nbar\" = \"v\" }",
		"{ .\"foo\rbar\" = \"v\" }",
		"{ .\"foo\vbar\" = \"v\" }",
		"{ .\"foo\fbar\" = \"v\" }",
		"{ .\" x\" = \"v\" }", // NBSP
	}

	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			pass1, err := ParseNoOptimizations(q)
			require.NoError(t, err)

			pass2, err := ParseNoOptimizations(pass1.String())
			require.NoError(t, err, "round-trip failed: %q", pass1.String())
			require.Equal(t, pass1, pass2, "round-trip changed AST: %q -> %q", q, pass1.String())
		})
	}
}
