package ll

import (
	"bytes"
	"io"
	"strings"

	"github.com/olekukonko/ll/lx"
)

// Writer returns an io.Writer that logs every write operation at the given level.
// Useful for capturing Stdout/Stderr from external processes.
func (l *Logger) Writer(level lx.LevelType) io.Writer {
	return &logWriter{
		logger: l,
		level:  level,
	}
}

// logWriter implements io.Writer to bridge external streams to ll.Logger
type logWriter struct {
	logger *Logger
	level  lx.LevelType
	buf    bytes.Buffer // Buffer for incomplete lines
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Buffer handling for partial lines (streams often write byte-by-byte)
	w.buf.Write(p)

	// Process complete lines
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil { // No newline found, buffer remains
			w.buf.WriteString(line)
			break
		}

		// Clean and log the complete line
		msg := strings.TrimSuffix(line, "\n")
		msg = strings.TrimSuffix(msg, "\r")

		if msg != "" {
			w.logger.log(w.level, lx.ClassText, msg, nil, false)
		}
	}

	return len(p), nil
}
