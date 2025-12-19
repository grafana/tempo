package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasLinkTraversal(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{
			query:    `{name="gateway"}`,
			expected: false,
		},
		{
			query:    `{name="database"} ->> {name="backend"}`,
			expected: true,
		},
		{
			query:    `{name="gateway"} <<- {name="backend"}`,
			expected: true,
		},
		{
			query:    `{name="database"} &->> {name="backend"} &->> {name="gateway"}`,
			expected: true,
		},
		{
			query:    `{name="gateway"} &<<- {name="backend"} &<<- {name="database"}`,
			expected: true,
		},
		{
			query:    `{name="frontend"} >> {name="backend"}`,
			expected: false, // >> is descendant, not link traversal
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)
			require.Equal(t, tt.expected, expr.HasLinkTraversal())
		})
	}
}

func TestExtractLinkChain(t *testing.T) {
	tests := []struct {
		name                 string
		query                string
		expectedPhases       int
		firstIsLinkTo        bool
		expectedPhaseOrder   []string // Expected execution order (terminal first)
	}{
		{
			name:           "two hop link to",
			query:          `{name="database"} &->> {name="backend"}`,
			expectedPhases: 2,
			firstIsLinkTo:  true,
			expectedPhaseOrder: []string{"backend", "database"}, // Terminal (backend) first
		},
		{
			name:           "three hop link to",
			query:          `{name="database"} &->> {name="backend"} &->> {name="gateway"}`,
			expectedPhases: 3,
			firstIsLinkTo:  true,
			expectedPhaseOrder: []string{"gateway", "backend", "database"}, // Terminal (gateway) first
		},
		{
			name:           "two hop link from",
			query:          `{name="gateway"} &<<- {name="backend"}`,
			expectedPhases: 2,
			firstIsLinkTo:  false,
			expectedPhaseOrder: []string{"gateway", "backend"}, // Terminal (gateway) first
		},
		{
			name:           "three hop link from",
			query:          `{name="gateway"} &<<- {name="backend"} &<<- {name="database"}`,
			expectedPhases: 3,
			firstIsLinkTo:  false,
			expectedPhaseOrder: []string{"gateway", "backend", "database"}, // Terminal (gateway) first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)
			
			chain := expr.ExtractLinkChain()
			require.Len(t, chain, tt.expectedPhases)
			
			if len(chain) > 0 {
				require.Equal(t, tt.firstIsLinkTo, chain[0].IsLinkTo)
				require.True(t, chain[0].IsUnion) // All test queries use union operators
			}
			
			// Verify execution order (terminal first)
			if len(tt.expectedPhaseOrder) > 0 {
				require.Len(t, chain, len(tt.expectedPhaseOrder), "phase count mismatch")
				for i, expectedName := range tt.expectedPhaseOrder {
					conditionStr := chain[i].Conditions.String()
					require.Contains(t, conditionStr, expectedName, 
						"Phase %d should contain '%s', got: %s", i, expectedName, conditionStr)
				}
			}
		})
	}
}

func TestLinkOperatorParsing(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		operator Operator
	}{
		{
			name:     "link to",
			query:    `{name="a"} ->> {name="b"}`,
			operator: OpSpansetLinkTo,
		},
		{
			name:     "link from",
			query:    `{name="a"} <<- {name="b"}`,
			operator: OpSpansetLinkFrom,
		},
		{
			name:     "union link to",
			query:    `{name="a"} &->> {name="b"}`,
			operator: OpSpansetUnionLinkTo,
		},
		{
			name:     "union link from",
			query:    `{name="a"} &<<- {name="b"}`,
			operator: OpSpansetUnionLinkFrom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.query)
			require.NoError(t, err)
			
			// Check that the operator was parsed correctly
			require.True(t, expr.HasLinkTraversal())
			
			chain := expr.ExtractLinkChain()
			require.NotEmpty(t, chain)
			require.Equal(t, tt.operator, chain[0].Op)
		})
	}
}

func TestLinkOperatorString(t *testing.T) {
	tests := []struct {
		operator Operator
		expected string
	}{
		{OpSpansetLinkTo, "->>"},
		{OpSpansetLinkFrom, "<<-"},
		{OpSpansetUnionLinkTo, "&->>"},
		{OpSpansetUnionLinkFrom, "&<<-"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.operator.String())
		})
	}
}

