package lh

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/olekukonko/ll/lx"
)

var jsonBufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// fieldsMapPool pools map[string]interface{} to reduce allocations
var fieldsMapPool = sync.Pool{
	New: func() any {
		return make(map[string]interface{}, 8)
	},
}

// jsonOutputPool pools JsonOutput structs to reduce allocations
var jsonOutputPool = sync.Pool{
	New: func() any {
		return &JsonOutput{
			Fields: make(map[string]interface{}, 8),
		}
	},
}

// JSONHandler is a handler that outputs log entries as JSON objects.
// It formats log entries with timestamp, level, message, namespace, fields, and optional
// stack traces or dump segments, writing the result to the provided writer.
// Thread-safe with a mutex to protect concurrent writes.
type JSONHandler struct {
	writer  io.Writer  // Destination for JSON output
	timeFmt string     // Format for timestamp (default: RFC3339Nano)
	pretty  bool       // Enable pretty printing with indentation if true
	mu      sync.Mutex // Protects concurrent access to writer
}

// JsonOutput represents the JSON structure for a log entry.
// It includes all relevant log data, such as timestamp, level, message, and optional
// stack trace or dump segments, serialized as a JSON object.
type JsonOutput struct {
	Time      string                 `json:"ts"`     // Timestamp in specified format
	Level     string                 `json:"lvl"`    // Log level (e.g., "INFO")
	Class     string                 `json:"class"`  // Entry class (e.g., "Text", "Dump")
	Msg       string                 `json:"msg"`    // Log message
	Namespace string                 `json:"ns"`     // Namespace path
	Stack     []byte                 `json:"stack"`  // Stack trace (if present)
	Dump      []dumpSegment          `json:"dump"`   // Hex/ASCII dump segments (for ClassDump)
	Fields    map[string]interface{} `json:"fields"` // Custom fields
}

// dumpSegment represents a single segment of a hex/ASCII dump.
// Used for ClassDump entries to structure position, hex values, and ASCII representation.
type dumpSegment struct {
	Offset int      `json:"offset"` // Starting byte offset of the segment
	Hex    []string `json:"hex"`    // Hexadecimal values of bytes
	ASCII  string   `json:"ascii"`  // ASCII representation of bytes
}

// NewJSONHandler creates a new JSONHandler writing to the specified writer.
// It initializes the handler with a default timestamp format (RFC3339Nano) and optional
// configuration functions to customize settings like pretty printing.
// Example:
//
//	handler := NewJSONHandler(os.Stdout)
//	logger := ll.New("app").Enable().Handler(handler)
//	logger.Info("Test") // Output: {"ts":"...","lvl":"INFO","class":"Text","msg":"Test","ns":"app","stack":null,"dump":null,"fields":null}
func NewJSONHandler(w io.Writer, opts ...func(*JSONHandler)) *JSONHandler {
	h := &JSONHandler{
		writer:  w,                // Set output writer
		timeFmt: time.RFC3339Nano, // Default timestamp format
	}
	// Apply configuration options
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Handle processes a log entry and writes it as JSON.
// It delegates to specialized methods based on the entry's class (Dump or regular),
// ensuring thread-safety with a mutex.
// Returns an error if JSON encoding or writing fails.
// Example:
//
//	handler.Handle(&lx.Entry{Message: "test", Level: lx.LevelInfo}) // Writes JSON object
func (h *JSONHandler) Handle(e *lx.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Handle dump entries separately
	if e.Class == lx.ClassDump {
		return h.handleDump(e)
	}
	// Handle standard log entries
	return h.handleRegular(e)
}

// Output sets the Writer destination for JSONHandler's output, ensuring thread safety with a mutex lock.
func (h *JSONHandler) Output(w io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.writer = w
}

// handleRegular handles standard log entries (non-dump).
// It converts the entry to a JsonOutput struct and encodes it as JSON,
// applying pretty printing if enabled. Logs encoding errors to stderr for debugging.
// Returns an error if encoding or writing fails.
// Example (internal usage):
//
//	h.handleRegular(&lx.Entry{Message: "test", Level: lx.LevelInfo}) // Writes JSON object
func (h *JSONHandler) handleRegular(e *lx.Entry) error {
	// Get fieldsMap from pool to avoid allocation
	fieldsMap := fieldsMapPool.Get().(map[string]interface{})
	// Clear any existing keys from previous use
	for k := range fieldsMap {
		delete(fieldsMap, k)
	}
	// Convert ordered fields to map for JSON output
	for _, pair := range e.Fields {
		fieldsMap[pair.Key] = pair.Value
	}

	// Get JsonOutput from pool
	entry := jsonOutputPool.Get().(*JsonOutput)
	entry.Time = e.Timestamp.Format(h.timeFmt)
	entry.Level = e.Level.String()
	entry.Class = e.Class.String()
	entry.Msg = e.Message
	entry.Namespace = e.Namespace
	entry.Dump = nil
	entry.Fields = fieldsMap
	entry.Stack = e.Stack

	// Acquire buffer from pool to avoid allocation and reduce syscalls
	buf := jsonBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		// Return all pooled objects
		jsonBufPool.Put(buf)
		// Reset and return fieldsMap to pool
		for k := range entry.Fields {
			delete(entry.Fields, k)
		}
		fieldsMapPool.Put(entry.Fields)
		// Reset and return JsonOutput to pool
		entry.Fields = nil
		entry.Stack = nil
		entry.Dump = nil
		jsonOutputPool.Put(entry)
	}()

	// Create JSON encoder writing to buffer (uses go-json for 2-5x speedup)
	enc := json.NewEncoder(buf)
	if h.pretty {
		// Enable indentation for pretty printing
		enc.SetIndent("", "  ")
	}
	// Encode JSON to buffer
	err := enc.Encode(entry)
	if err != nil {
		// Log encoding error for debugging
		fmt.Fprintf(os.Stderr, "JSON encode error: %v\n", err)
		return err
	}
	// Write buffer to underlying writer in one go
	_, err = h.writer.Write(buf.Bytes())
	return err
}

// handleDump processes ClassDump entries, converting hex dump output to JSON segments.
// It parses the dump message into structured segments with offset, hex, and ASCII data,
// encoding them as a JsonOutput struct.
// Returns an error if parsing or encoding fails.
// Example (internal usage):
//
//	h.handleDump(&lx.Entry{Class: lx.ClassDump, Message: "pos 00 hex: 61 62 'ab'"}) // Writes JSON with dump segments
func (h *JSONHandler) handleDump(e *lx.Entry) error {
	var segments []dumpSegment
	lines := strings.Split(e.Message, "\n")
	// Parse each line of the dump message
	for _, line := range lines {
		if !strings.HasPrefix(line, "pos") {
			continue // Skip non-dump lines
		}
		parts := strings.SplitN(line, "hex:", 2)
		if len(parts) != 2 {
			continue // Skip invalid lines
		}
		// Parse position
		var offset int
		fmt.Sscanf(parts[0], "pos %d", &offset)
		// Parse hex and ASCII
		hexAscii := strings.SplitN(parts[1], "'", 2)
		hexStr := strings.Fields(strings.TrimSpace(hexAscii[0]))
		// Create dump segment
		segments = append(segments, dumpSegment{
			Offset: offset,                         // Set byte offset
			Hex:    hexStr,                         // Set hex values
			ASCII:  strings.Trim(hexAscii[1], "'"), // Set ASCII representation
		})
	}

	// Get fieldsMap from pool
	fieldsMap := fieldsMapPool.Get().(map[string]interface{})
	for k := range fieldsMap {
		delete(fieldsMap, k)
	}
	for _, pair := range e.Fields {
		fieldsMap[pair.Key] = pair.Value
	}

	// Get JsonOutput from pool
	entry := jsonOutputPool.Get().(*JsonOutput)
	entry.Time = e.Timestamp.Format(h.timeFmt)
	entry.Level = e.Level.String()
	entry.Class = e.Class.String()
	entry.Msg = "dumping segments"
	entry.Namespace = e.Namespace
	entry.Dump = segments
	entry.Fields = fieldsMap
	entry.Stack = e.Stack

	// Acquire buffer from pool
	buf := jsonBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		jsonBufPool.Put(buf)
		for k := range entry.Fields {
			delete(entry.Fields, k)
		}
		fieldsMapPool.Put(entry.Fields)
		entry.Fields = nil
		entry.Stack = nil
		entry.Dump = nil
		jsonOutputPool.Put(entry)
	}()

	// Encode JSON output with dump segments to buffer
	enc := json.NewEncoder(buf)
	if h.pretty {
		enc.SetIndent("", "  ")
	}
	err := enc.Encode(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON dump encode error: %v\n", err)
		return err
	}
	// Write buffer to underlying writer
	_, err = h.writer.Write(buf.Bytes())
	return err
}
