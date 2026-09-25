package compare

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark"
)

func TestNaming(t *testing.T) {
	withReadBuffer := func(size int) *benchmark.Result {
		r := newResult("mac")
		r.Options.ReadBufferSize = size
		return r
	}
	run := func(name, file string, r *benchmark.Result) Run {
		return Run{Name: name, File: file, Result: r}
	}

	tests := []struct {
		name      string
		runs      []Run
		names     []string
		namedBy   []string
		orderedBy string
		baseline  int
	}{
		{
			name: "one number varies: named by its value and put in its order",
			runs: []Run{
				run("", "b.json", withReadBuffer(4<<20)),
				run("", "c.json", withReadBuffer(16<<20)),
				run("", "a.json", withReadBuffer(0)),
			},
			names:     []string{"default", "4MiB", "16MiB"},
			namedBy:   []string{"readBufferSize"},
			orderedBy: "readBufferSize",
			baseline:  1, // the run given first, wherever it lands
		},
		{
			name: "several settings vary: named field by field, left in order",
			runs: func() []Run {
				r := withReadBuffer(4 << 20)
				r.Options.ReadBufferCount = 8
				return []Run{run("", "a.json", withReadBuffer(0)), run("", "b.json", r)}
			}(),
			names:   []string{"readBufferSize=default readBufferCount=default", "readBufferSize=4MiB readBufferCount=8"},
			namedBy: []string{"readBufferSize", "readBufferCount"},
		},
		{
			name: "the build varies: named by short SHA",
			runs: func() []Run {
				r := newResult("mac")
				r.RunEnv.GitSHA = "0123456789abcdef"
				return []Run{run("", "a.json", newResult("mac")), run("", "b.json", r)}
			}(),
			names:   []string{"abc", "01234567"},
			namedBy: []string{"gitSHA"},
		},
		{
			name: "only where they ran varies: named by that",
			runs: func() []Run {
				r := newResult("mac")
				r.RunEnv.GoVersion = "go1.26"
				return []Run{run("", "a.json", newResult("mac")), run("", "b.json", r)}
			}(),
			names:   []string{"go1.27", "go1.26"},
			namedBy: []string{"goVersion"},
		},
		{
			name: "an accidental difference in where they ran stays out of the names",
			runs: []Run{
				run("", "a.json", withReadBuffer(0)),
				run("", "b.json", func() *benchmark.Result { r := withReadBuffer(4 << 20); r.RunEnv.Hostname = "linux"; return r }()),
			},
			names:     []string{"default", "4MiB"},
			namedBy:   []string{"readBufferSize"},
			orderedBy: "readBufferSize",
		},
		{
			name:  "nothing varies: named after their files",
			runs:  []Run{run("", "out/first.json", newResult("mac")), run("", "out/second.json", newResult("mac"))},
			names: []string{"first", "second"},
		},
		{
			name:  "files named alike are numbered",
			runs:  []Run{run("", "a/result.json", newResult("mac")), run("", "b/result.json", newResult("mac"))},
			names: []string{"result", "result #2"},
		},
		{
			name: "repeats of one setup are named after their files",
			runs: []Run{
				run("", "a.json", withReadBuffer(0)),
				run("", "b.json", withReadBuffer(0)),
				run("", "c.json", withReadBuffer(4<<20)),
			},
			names:     []string{"a", "b", "4MiB"},
			namedBy:   []string{"readBufferSize"},
			orderedBy: "readBufferSize",
		},
		{
			name: "explicit names are kept, and a found name gives way to them",
			runs: []Run{
				run("4MiB", "a.json", withReadBuffer(0)),
				run("", "b.json", withReadBuffer(4<<20)),
			},
			names:     []string{"4MiB", "b"},
			orderedBy: "readBufferSize",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.runs)
			require.NoError(t, err)
			require.Equal(t, tt.names, c.Names())
			require.Equal(t, tt.namedBy, c.NamedBy)
			require.Equal(t, tt.orderedBy, c.OrderedBy)
			require.Equal(t, tt.baseline, c.Baseline)
		})
	}
}

func TestNamingKeepsCasesWithTheirRuns(t *testing.T) {
	small := newResult("mac", caseResult("search/nopredicate", 1, nil))
	small.Options.ReadBufferSize = 1 << 20
	large := newResult("mac", caseResult("search/nopredicate", 2, nil))
	large.Options.ReadBufferSize = 8 << 20

	c, err := New([]Run{{File: "large.json", Result: large}, {File: "small.json", Result: small}})
	require.NoError(t, err)
	require.Equal(t, []string{"1MiB", "8MiB"}, c.Names())
	require.Equal(t, 1, c.Baseline)

	// Reordering the runs moves their results and settings with them.
	require.Equal(t, int64(1), c.Cases[0].Results[0].Matched)
	require.Equal(t, int64(2), c.Cases[0].Results[1].Matched)
	require.Equal(t, []Change{{Field: "readBufferSize", From: "8MiB", To: "1MiB", Kind: Setup}}, changes(c.settings[c.Baseline], c.settings[0]))
}

func TestNamingRejectsDuplicateExplicitNames(t *testing.T) {
	_, err := New([]Run{{Name: "a", Result: newResult("mac")}, {Name: "a", Result: newResult("mac")}})
	require.Error(t, err)
}
