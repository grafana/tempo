package log

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestSlogFromGoKit(t *testing.T) {
	var buf bytes.Buffer
	kit := kitlog.NewLogfmtLogger(&buf)
	logLevel = levelInfo

	logger := SlogFromGoKit(kit)
	require.NotNil(t, logger)

	logger.Info("hello", "k", "v")
	require.Contains(t, buf.String(), "hello")
	require.Contains(t, buf.String(), "k=v")

	// debug should be filtered at info level
	buf.Reset()
	logger.Log(t.Context(), slog.LevelDebug, "hidden")
	require.Empty(t, buf.String())
}

func TestSlogFromGoKitNoDuplicateTSOrCaller(t *testing.T) {
	var buf bytes.Buffer
	base := kitlog.NewLogfmtLogger(&buf)
	goKitBase = base
	// Simulate InitLogger enriching the global Logger with ts/caller.
	Logger = kitlog.With(base, "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.Caller(3))
	logLevel = levelInfo

	logger := SlogFromGoKit(Logger)
	logger.Info("hello", "k", "v")
	out := buf.String()

	require.Contains(t, out, "hello")
	require.Contains(t, out, "k=v")
	// slog-gokit emits time=; go-kit InitLogger uses ts=. Using the base
	// logger means we should not also get ts= from the enriched Logger.
	require.NotContains(t, out, "ts=")
	require.Equal(t, 1, strings.Count(out, "caller="), "caller should appear once, not duplicated")
}
