package app

import (
	"math"
	"os"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitAutoMemLimit(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		ratio       float64
		gomemlimit  *string
		wantChanged bool
	}{
		{name: "disabled", enabled: false, ratio: 0.8},
		{name: "enabled", enabled: true, ratio: 0.8, wantChanged: true},
		{name: "enabled with invalid ratio", enabled: true, ratio: 0},
		{name: "enabled with GOMEMLIMIT set", enabled: true, ratio: 0.8, gomemlimit: new("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// the memory limit is process-wide, so restore it and start from "no limit"
			prev := debug.SetMemoryLimit(math.MaxInt64)
			t.Cleanup(func() { debug.SetMemoryLimit(prev) })

			// t.Setenv restores the original values on cleanup
			t.Setenv("AUTOMEMLIMIT", "")
			require.NoError(t, os.Unsetenv("AUTOMEMLIMIT"))
			t.Setenv("GOMEMLIMIT", "")
			require.NoError(t, os.Unsetenv("GOMEMLIMIT"))
			if tt.gomemlimit != nil {
				t.Setenv("GOMEMLIMIT", *tt.gomemlimit)
			}

			cfg := &Config{}
			cfg.Memory.AutoMemLimitEnabled = tt.enabled
			cfg.Memory.AutoMemLimitRatio = tt.ratio

			InitAutoMemLimit(cfg)

			got := debug.SetMemoryLimit(-1)
			if tt.wantChanged {
				require.Greater(t, got, int64(0))
				require.Less(t, got, int64(math.MaxInt64))
			} else {
				require.Equal(t, int64(math.MaxInt64), got)
			}
		})
	}
}
