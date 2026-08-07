package log

import (
	"log/slog"

	"github.com/go-kit/log"
	slgk "github.com/tjhop/slog-gokit"
)

const (
	levelInfo  = "info"
	levelWarn  = "warn"
	levelError = "error"
	levelDebug = "debug"
)

// SlogFromGoKit returns a slog.Logger backed by the given go-kit logger.
// This is the first step toward replacing go-kit/log with log/slog (#4819):
// new code can take *slog.Logger while existing call sites keep using go-kit
// until they are migrated. dskit still expects go-kit.Logger on server.Config.
//
// Do not pass Logger from InitLogger: that logger already has "ts" and
// "caller", and slog-gokit adds those attributes itself. Pass a bare logger
// (for example from tests), or nil / Logger to use the process-wide base
// logger that omits those fields.
func SlogFromGoKit(logger log.Logger) *slog.Logger {
	// Using the global Logger would duplicate ts/caller that slog-gokit adds.
	if logger == nil || logger == Logger {
		logger = goKitBase
	}

	var sl slog.Level
	switch logLevel {
	case levelInfo:
		sl = slog.LevelInfo
	case levelWarn:
		sl = slog.LevelWarn
	case levelError:
		sl = slog.LevelError
	default:
		sl = slog.LevelDebug
	}

	lvl := slog.LevelVar{}
	lvl.Set(sl)
	return slog.New(slgk.NewGoKitHandler(logger, &lvl))
}
