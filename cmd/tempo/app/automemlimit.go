package app

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

// InitAutoMemLimit configures Go's memory limit using automemlimit based on the
// provided config. It is exported so that downstream consumers can call it.
func InitAutoMemLimit(config *Config) {
	if !config.Memory.AutoMemLimitEnabled {
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	limit, err := memlimit.SetGoMemLimitWithOpts(
		memlimit.WithRatio(config.Memory.AutoMemLimitRatio),
		memlimit.WithProvider(
			memlimit.ApplyFallback(
				memlimit.FromCgroup,
				memlimit.FromSystem,
			),
		),
		memlimit.WithRefreshInterval(15*time.Second),
		memlimit.WithLogger(logger),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to set GOMEMLIMIT: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "info: GOMEMLIMIT set to %d bytes (ratio: %.2f)\n", limit, config.Memory.AutoMemLimitRatio)
}
