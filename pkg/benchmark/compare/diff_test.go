package compare

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark"
)

func TestSettings(t *testing.T) {
	r := &benchmark.Result{
		RunEnv: benchmark.RunEnv{GitSHA: "3109151e0dba619a74c79d14f161437ff43903ae", GoVersion: "go1.27.1", GoMaxProcs: 12, Hostname: "mac"},
		Options: benchmark.RunOptions{
			Repeat: 1, TargetBytesPerRequest: 100 << 20, SearchLimit: 20, MaxSeries: 1000,
			ReadBufferSize: 4 << 20,
		},
		Shards: 55,
	}
	ss, err := settings(r)
	require.NoError(t, err)

	type fv struct {
		field, value string
		kind         Kind
	}
	got := make([]fv, len(ss))
	for i, s := range ss {
		got[i] = fv{s.Field, s.Value, s.Kind}
	}
	// Options come in the order they are declared, leaving out those left at
	// default, then the build, where the run happened, and what followed.
	require.Equal(t, []fv{
		{"repeat", "1", Setup},
		{"targetBytesPerRequest", "100MiB", Setup},
		{"searchLimit", "20", Setup},
		{"maxSeries", "1000", Setup},
		{"exemplars", "0", Setup},
		{"readBufferSize", "4MiB", Setup},
		{"tempoVersion", "unknown", Build},
		{"gitSHA", "3109151e", Build},
		{"goVersion", "go1.27.1", Environment},
		{"goMaxProcs", "12", Environment},
		{"hostname", "mac", Environment},
		{"shards", "55", Derived},
	}, got)

	require.True(t, ss[5].isNumber)
	require.Equal(t, float64(4<<20), ss[5].number)
	require.False(t, ss[7].isNumber)
}

func TestFormatIBytes(t *testing.T) {
	tests := []struct {
		n    uint64
		want string
	}{
		{0, "0B"},
		{512, "512B"},
		{1 << 10, "1KiB"},
		{4 << 20, "4MiB"},
		{100 << 20, "100MiB"},
		{1 << 30, "1GiB"},
		{1536, "1.5KiB"},
		{1_500_000, "1.4MiB"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, formatIBytes(tt.n), "%d bytes", tt.n)
	}
}

func TestChanges(t *testing.T) {
	base := newResult("mac")
	run := newResult("mac")
	run.Options.ReadBufferSize = 4 << 20
	run.Options.TargetBytesPerRequest = 50 << 20
	run.RunEnv.GoMaxProcs = 8
	run.Shards = 110

	baseSettings, err := settings(base)
	require.NoError(t, err)
	runSettings, err := settings(run)
	require.NoError(t, err)

	got := changes(baseSettings, runSettings)
	strs := make([]string, len(got))
	for i, ch := range got {
		strs[i] = ch.String()
	}
	require.Equal(t, []string{
		"targetBytesPerRequest 0B → 50MiB",
		"readBufferSize default → 4MiB",
		"goMaxProcs 12 → 8",
		"shards 55 → 110",
	}, strs)
	require.Equal(t, Derived, got[3].Kind)

	// The other way round, an option the baseline set reads as back to default.
	got = changes(runSettings, baseSettings)
	require.Equal(t, Change{Field: "readBufferSize", From: "4MiB", To: "default", Kind: Setup}, got[1])

	require.Empty(t, changes(baseSettings, baseSettings))
}

func TestBaselineSettings(t *testing.T) {
	base := newResult("mac")
	other := newResult("linux")
	other.Options.ReadBufferSize = 4 << 20
	c, err := New([]Run{{Name: "base", Result: base}, {Name: "other", Result: other}})
	require.NoError(t, err)

	// What the others are measured from: the settings that vary first, then
	// the baseline's build and where it ran.
	var got []string
	for _, s := range c.BaselineSettings(0) {
		got = append(got, s.String())
	}
	require.Equal(t, []string{
		"readBufferSize default",
		"hostname mac",
		"gitSHA abc",
		"goVersion go1.27",
		"goMaxProcs 12",
	}, got)

	require.Equal(t, []Change{
		{Field: "readBufferSize", From: "default", To: "4MiB", Kind: Setup},
		{Field: "hostname", From: "mac", To: "linux", Kind: Environment},
	}, c.Changes(1, 0))
}
