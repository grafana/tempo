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
