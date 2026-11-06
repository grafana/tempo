package frontend

import (
	"flag"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchSharderConfigDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	assert.Equal(t, uint32(256*1024), cfg.Search.Sharder.MaxLimit)
}

func TestQueryTrackerMaxQueryTextBytesFlag(t *testing.T) {
	const flagName = "-query-frontend.query-tracker.max-query-text-bytes"
	for _, tc := range []struct {
		name    string
		args    []string
		want    int
		wantErr string
	}{
		{name: "zero disables truncation", args: []string{flagName + "=0"}, want: 0},
		{name: "positive limit", args: []string{flagName + "=128"}, want: 128},
		{name: "non-numeric value", args: []string{flagName + "=invalid"}, wantErr: "invalid value"},
		{name: "fractional value", args: []string{flagName + "=1.5"}, wantErr: "invalid value"},
		{name: "empty value", args: []string{flagName + "="}, wantErr: "invalid value"},
		{name: "integer overflow", args: []string{flagName + "=18446744073709551616"}, wantErr: "invalid value"},
		{name: "missing value", args: []string{flagName}, wantErr: "flag needs an argument"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			cfg := &Config{}
			cfg.RegisterFlagsAndApplyDefaults("query-frontend", fs)
			err := fs.Parse(tc.args)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.QueryTracker.MaxQueryTextBytes)
		})
	}
}

func TestQueryTrackerConfig(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("query-frontend", fs)

	assert.False(t, cfg.QueryTracker.Enabled)
	assert.Equal(t, 100, cfg.QueryTracker.MaxQueriesPerTenant)
	assert.Equal(t, 4096, cfg.QueryTracker.MaxQueryTextBytes)
	assert.NoError(t, fs.Parse([]string{"-query-frontend.query-tracker.enabled=true", "-query-frontend.query-tracker.max-queries-per-tenant=3"}))
	assert.True(t, cfg.QueryTracker.Enabled)
	assert.Equal(t, 3, cfg.QueryTracker.MaxQueriesPerTenant)
}
