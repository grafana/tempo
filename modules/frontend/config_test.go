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

func TestActiveQueriesEndpointConfig(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("query-frontend", fs)

	assert.False(t, cfg.ActiveQueriesEnabled)
	assert.NoError(t, fs.Parse([]string{"-query-frontend.active-queries-enabled=true"}))
	assert.True(t, cfg.ActiveQueriesEnabled)
}
