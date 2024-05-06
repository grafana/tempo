package traceql

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
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
				pass1, err := Parse(q)
				require.NoError(t, err)

				// now parse it a second time and confirm that it parses the same way twice
				pass2, err := Parse(pass1.String())
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
