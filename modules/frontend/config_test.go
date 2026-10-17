package frontend

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchSharderConfigDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	assert.Equal(t, uint32(256*1024), cfg.Search.Sharder.MaxLimit)
}

func TestQueryTrackerConfig(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("query-frontend", fs)

	assert.False(t, cfg.QueryTracker.Enabled)
	assert.Equal(t, 20, cfg.QueryTracker.MaxQueriesPerTenant)
	assert.Equal(t, 4096, cfg.QueryTracker.MaxQueryTextBytes)
	assert.NoError(t, fs.Parse([]string{"-query-frontend.query-tracker.enabled=true", "-query-frontend.query-tracker.max-queries-per-tenant=3"}))
	assert.True(t, cfg.QueryTracker.Enabled)
	assert.Equal(t, 3, cfg.QueryTracker.MaxQueriesPerTenant)
}
