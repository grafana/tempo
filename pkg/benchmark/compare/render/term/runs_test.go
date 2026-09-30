package term

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
)

func TestRuns(t *testing.T) {
	base := comparetest.Result("mac")
	other := comparetest.Result("linux")
	other.Options.ReadBufferSize = 4 << 20
	c, err := compare.New([]compare.Run{{Name: "base", Result: base}, {Name: "4MiB", Result: other}})
	require.NoError(t, err)

	lines := Runs(render.NewRuns(c, 0))
	require.Equal(t, "base  baseline · readBufferSize default · hostname mac · gitSHA abc · goVersion go1.27 · goMaxProcs 12", lines[0].String())
	require.Equal(t, "4MiB  readBufferSize default → 4MiB · ⚠ hostname mac → linux", lines[1].String())

	// The lead is shown as the run, and the details as they read.
	require.Equal(t, render.Text{Text: "4MiB  ", Style: render.RunName, Run: 1}, lines[1][0])
	require.Equal(t, render.Text{Text: " · ", Style: render.Dim}, lines[1][2])
	require.Equal(t, render.Warn, lines[1][3].Style)

	// Runs numbered to head columns are listed with their numbers, as the key.
	c.Runs[1].Name = "a-longer-name"
	lines = Runs(render.NewRuns(c, 0))
	require.Equal(t, "#1 base           ", lines[0][0].Text)
	require.Equal(t, "#2 a-longer-name  ", lines[1][0].Text)
}
