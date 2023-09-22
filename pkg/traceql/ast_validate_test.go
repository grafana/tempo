package traceql

import (
	"errors"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type TestQueries struct {
	Valid         []string `yaml:"valid"`
	ParseFails    []string `yaml:"parse_fails"`
	ValidateFails []string `yaml:"validate_fails"`
	Unsupported   []string `yaml:"unsupported"`
	Dump          []string `yaml:"dump"`
}

func TestExamples(t *testing.T) {
	b, err := os.ReadFile(testExamplesFile)
	require.NoError(t, err)

	queries := &TestQueries{}
	err = yaml.Unmarshal(b, queries)
	require.NoError(t, err)

	for _, q := range queries.Valid {
		t.Run("valid - "+q, func(t *testing.T) {
			p, err := Parse(q)
			require.NoError(t, err)
			err = p.validate()
			require.NoError(t, err)
		})
	}

	for _, q := range queries.ParseFails {
		t.Run("parse fails - "+q, func(t *testing.T) {
			_, err := Parse(q)
			require.Error(t, err)
		})
	}

	for _, q := range queries.ValidateFails {
		t.Run("validate fails - "+q, func(t *testing.T) {
			p, err := Parse(q)
			require.NoError(t, err)
			err = p.validate()
			require.Error(t, err)
			var unErr *unsupportedError
			require.False(t, errors.As(err, &unErr))
		})
	}

	for _, q := range queries.Unsupported {
		t.Run("unsupported - "+q, func(t *testing.T) {
			p, err := Parse(q)
			require.NoError(t, err)
			err = p.validate()
			require.Error(t, err)
			var unErr *unsupportedError
			require.True(t, errors.As(err, &unErr))
		})
	}

	scs := spew.ConfigState{DisableMethods: true, Indent: " "}
	for _, q := range queries.Dump {
		t.Run("dump - "+q, func(t *testing.T) {
			yyDebug = 3
			p, err := Parse(q)
			yyDebug = 0
			require.NoError(t, err)
			scs.Dump(p)
		})
	}
}
