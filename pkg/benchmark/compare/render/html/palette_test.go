package html

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
)

func TestColor(t *testing.T) {
	require.Empty(t, color(render.Text{Style: render.Plain}), "plain text has no colour")
	require.Empty(t, color(render.Text{Style: render.Title}), "a title stands out by weight")

	better := color(render.Text{Style: render.Better})
	require.Equal(t, better, color(render.Text{Style: render.MuchBetter}), "how far a change went shows in weight, not colour")
	require.NotEqual(t, better, color(render.Text{Style: render.Worse}))

	// Runs have a colour each, in turn, whether or not they are the baseline.
	first := color(render.Text{Style: render.RunName, Run: 0})
	require.NotEqual(t, first, color(render.Text{Style: render.RunName, Run: 1}))
	require.Equal(t, first, color(render.Text{Style: render.BaselineName, Run: 0}))
	require.Equal(t, first, color(render.Text{Style: render.RunName, Run: len(runColors)}))
}
