package log

import (
	"os"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	dslog "github.com/grafana/dskit/log"
)

// Logger is a shared go-kit logger.
// TODO: Change all components to take a non-global logger via their constructors.
// Prefer accepting a non-global logger as an argument.
var Logger = kitlog.NewNopLogger()

// InitLogger initialises the global gokit logger and returns that logger.
func InitLogger(logFormat string, logLevel dslog.Level) kitlog.Logger {
	writer := kitlog.NewSyncWriter(os.Stderr)
	logger := dslog.NewGoKitWithWriter(logFormat, writer)

	// use UTC timestamps and skip 5 stack frames.
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.Caller(5))

	// Must put the level filter last for efficiency.
	logger = level.NewFilter(logger, logLevel.Option)

	Logger = logger
	return logger
}
