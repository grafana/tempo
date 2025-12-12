package drain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogCluster_tokenDistance_ExactMatch(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "/users"},
		ParamString: "<_>",
	}

	similarity, paramCount := cluster.tokenDistance([]string{"GET", "/users"})

	assert.Equal(t, 1.0, similarity)
	assert.Equal(t, 0, paramCount)
}

func TestLogCluster_tokenDistance_ParamMatch(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "<_>"},
		ParamString: "<_>",
	}

	similarity, paramCount := cluster.tokenDistance([]string{"GET", "/users"})

	// Similarity is based on matching tokens, not params
	// GET matches, <_> doesn't match /users (it's a param)
	assert.Equal(t, 0.5, similarity) // 1 out of 2 tokens match
	assert.Equal(t, 1, paramCount)
}

func TestLogCluster_tokenDistance_Mismatch(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "/users"},
		ParamString: "<_>",
	}

	similarity, paramCount := cluster.tokenDistance([]string{"POST", "/users"})

	assert.Equal(t, 0.5, similarity) // 1 out of 2 tokens match (/users)
	assert.Equal(t, 0, paramCount)
}

func TestLogCluster_tokenDistance_MultipleParams(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "<_>", "/api", "<_>"},
		ParamString: "<_>",
	}

	similarity, paramCount := cluster.tokenDistance([]string{"GET", "/users", "/api", "123"})

	assert.Equal(t, 0.5, similarity) // 2 out of 4 tokens match (GET, /api)
	assert.Equal(t, 2, paramCount)
}

func TestLogCluster_tokenDistance_NoMatch(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "/users"},
		ParamString: "<_>",
	}

	similarity, paramCount := cluster.tokenDistance([]string{"POST", "/posts"})

	assert.Equal(t, 0.0, similarity) // 0 out of 2 tokens match
	assert.Equal(t, 0, paramCount)
}

func TestLogCluster_tokenDistance_DifferentLengthPanics(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "/users"},
		ParamString: "<_>",
	}

	require.Panics(t, func() {
		cluster.tokenDistance([]string{"GET", "/users", "/extra"})
	})
}

func TestLogCluster_ingestTokens_NoChange(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "/users"},
		ParamString: "<_>",
		Size:        1,
	}

	cluster.ingestTokens([]string{"GET", "/users"})

	assert.Equal(t, []string{"GET", "/users"}, cluster.Tokens)
	assert.Equal(t, 2, cluster.Size)
}

func TestLogCluster_ingestTokens_PatternMerging(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "/users"},
		ParamString: "<_>",
		Size:        1,
		cache:       "GET/users", // cached string
	}

	cluster.ingestTokens([]string{"GET", "/posts"})

	// Different token should be replaced with param string
	assert.Equal(t, []string{"GET", "<_>"}, cluster.Tokens)
	assert.Equal(t, 2, cluster.Size)
	// Cache should be cleared when tokens change
	assert.Empty(t, cluster.cache)
}

func TestLogCluster_ingestTokens_MultipleMerges(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "/users", "/api"},
		ParamString: "<_>",
		Size:        1,
	}

	// First merge: different second token
	cluster.ingestTokens([]string{"GET", "/posts", "/api"})
	assert.Equal(t, []string{"GET", "<_>", "/api"}, cluster.Tokens)

	// Second merge: different third token
	cluster.ingestTokens([]string{"GET", "/users", "/v2"})
	assert.Equal(t, []string{"GET", "<_>", "<_>"}, cluster.Tokens)
	assert.Equal(t, 3, cluster.Size)
}

func TestLogCluster_ingestTokens_ParamAlreadySet(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "<_>"},
		ParamString: "<_>",
		Size:        1,
	}

	// Ingesting any value for a param slot should keep it as param
	cluster.ingestTokens([]string{"GET", "/users"})
	assert.Equal(t, []string{"GET", "<_>"}, cluster.Tokens)
	assert.Equal(t, 2, cluster.Size)
}

func TestLogCluster_ingestTokens_DifferentLengthPanics(t *testing.T) {
	t.Parallel()

	cluster := &LogCluster{
		Tokens:      []string{"GET", "/users"},
		ParamString: "<_>",
	}

	require.Panics(t, func() {
		cluster.ingestTokens([]string{"GET"})
	})
}

func TestLogCluster_String_CacheClearedOnIngest(t *testing.T) {
	t.Parallel()

	joinFunc := func(tokens []string) string {
		return strings.Join(tokens, " ")
	}

	cluster := &LogCluster{
		Tokens:      []string{"GET", "/users"},
		ParamString: "<_>",
		Stringer:    joinFunc,
		cache:       "GET /users",
	}

	// Cache should be set
	assert.Equal(t, "GET /users", cluster.cache)

	// Ingesting different tokens should clear cache
	cluster.ingestTokens([]string{"GET", "/posts"})
	assert.Empty(t, cluster.cache)

	// Next String() call should recompute
	result := cluster.String()
	assert.Equal(t, "GET <_>", result)
	assert.Equal(t, "GET <_>", cluster.cache)
}
