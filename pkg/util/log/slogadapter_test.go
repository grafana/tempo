package log

import (
	"bytes"
	"log/slog"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestSlogFromGoKit(t *testing.T) {
	var buf bytes.Buffer
	kit := kitlog.NewLogfmtLogger(&buf)
	logLevel = "info"

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
