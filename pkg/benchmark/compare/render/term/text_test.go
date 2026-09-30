package term

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
)

func TestPaint(t *testing.T) {
	l := Line{{Text: "plain"}, {Text: "dim", Style: render.Dim}, {Text: "base", Style: render.BaselineName, Run: 2}}
	require.Equal(t, "plaindimbase", l.String())
	require.Equal(t,
		Style(render.Plain).Render("plain")+Style(render.Dim).Render("dim")+textStyle(l[2]).Render("base"),
		Paint(l))
}

func TestStyles(t *testing.T) {
	// A run is shown in its colour, and the baseline in bold as well.
	run := render.Text{Style: render.BaselineName, Run: 2}
	c, ok := run.Color()
	require.True(t, ok)
	require.Equal(t, lipgloss.Color(c.ANSI), textStyle(run).GetForeground())
	require.True(t, textStyle(run).GetBold())

	require.Equal(t, lipgloss.NoColor{}, Style(render.Plain).GetForeground(), "plain text is left as it is")
	require.False(t, Style(render.Plain).GetBold())
	require.False(t, Style(render.Dim).GetBold())
	require.True(t, Style(render.Title).GetBold())
	require.True(t, Style(render.MuchWorse).GetBold())
	require.NotEqual(t, Style(render.Better).GetForeground(), Style(render.Worse).GetForeground())
}

func TestClip(t *testing.T) {
	require.Equal(t, "short", Clip("short", 10))
	require.Equal(t, "cut …", Clip("cut here", 5))
	require.Equal(t, "", Clip("anything", 0))
}
