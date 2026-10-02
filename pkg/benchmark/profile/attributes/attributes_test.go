package attributes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validAttributes() *Profiles {
	return &Profiles{
		Spans: 2000,
		Ranked: []Profile{
			{
				Scope: ScopeIntrinsic, Name: "duration", Type: TypeDuration,
				Dedicated: true, TotalBytes: 16000, Cardinality: 1900, Density: 1,
				Quantiles: []Quantile{{Q: 0.5, Value: "4s", Selectivity: 0.5}, {Q: 0.9, Value: "9s", Selectivity: 0.1}},
			},
			{
				Scope: ScopeResource, Name: "service.name", Type: TypeString,
				Dedicated: true, TotalBytes: 10000, Cardinality: 2, Density: 1,
				Values: []Value{{Value: "svc-a", Selectivity: 0.6}, {Value: "svc-b", Selectivity: 0.4}},
			},
		},
	}
}

func TestAttributeProfilesValidate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mutate  func(*Profiles)
		wantErr string
	}{
		{"no spans", func(a *Profiles) { a.Spans = 0 }, "cover 0 spans"},
		{
			name:    "not ranked by bytes",
			mutate:  func(a *Profiles) { a.Ranked[0], a.Ranked[1] = a.Ranked[1], a.Ranked[0] },
			wantErr: "not ranked by total bytes",
		},
		{"unknown scope", func(a *Profiles) { a.Ranked[1].Scope = "event" }, "unknown scope"},
		{"unknown type", func(a *Profiles) { a.Ranked[1].Type = "map" }, "unknown type"},
		{"density out of range", func(a *Profiles) { a.Ranked[1].Density = 1.5 }, "density"},
		{
			name:    "both values and quantiles",
			mutate:  func(a *Profiles) { a.Ranked[0].Values = []Value{{Value: "1s", Selectivity: 0.1}} },
			wantErr: "exactly one of values and quantiles",
		},
		{"neither values nor quantiles", func(a *Profiles) { a.Ranked[1].Values = nil }, "exactly one of values and quantiles"},
		{
			name: "quantiles on a string",
			mutate: func(a *Profiles) {
				a.Ranked[1].Values = nil
				a.Ranked[1].Quantiles = a.Ranked[0].Quantiles
			},
			wantErr: "only for numeric types",
		},
		{
			name:    "values not sorted",
			mutate:  func(a *Profiles) { v := a.Ranked[1].Values; v[0], v[1] = v[1], v[0] },
			wantErr: "not sorted by selectivity",
		},
		{"value selectivity out of range", func(a *Profiles) { a.Ranked[1].Values[1].Selectivity = -0.1 }, "not a fraction"},
		{"quantile out of range", func(a *Profiles) { a.Ranked[0].Quantiles[1].Q = 1 }, "not in (0, 1)"},
		{
			name:    "quantiles not increasing",
			mutate:  func(a *Profiles) { q := a.Ranked[0].Quantiles; q[0], q[1] = q[1], q[0] },
			wantErr: "increasing order",
		},
		{"quantile selectivity out of range", func(a *Profiles) { a.Ranked[0].Quantiles[0].Selectivity = 2 }, "not a fraction"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := validAttributes()
			tc.mutate(a)
			require.ErrorContains(t, a.Validate(), tc.wantErr)
		})
	}

	require.NoError(t, validAttributes().Validate())
}
