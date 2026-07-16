package log

import (
	"log/slog"

	"github.com/go-kit/log"
	slgk "github.com/tjhop/slog-gokit"
)

// SlogFromGoKit returns a slog.Logger backed by the given go-kit logger.
// This is the first step toward replacing go-kit/log with log/slog (#4819):
// new code can take *slog.Logger while existing call sites keep using go-kit
// until they are migrated. dskit still expects go-kit.Logger on server.Config.
func SlogFromGoKit(logger log.Logger) *slog.Logger {
	var sl slog.Level
	switch logLevel {
	case "info":
		sl = slog.LevelInfo
	case "warn":
		sl = slog.LevelWarn
	case "error":
		sl = slog.LevelError
	default:
		sl = slog.LevelDebug
	}

	lvl := slog.LevelVar{}
	lvl.Set(sl)
	return slog.New(slgk.NewGoKitHandler(logger, &lvl))
}
