package ll

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/olekukonko/cat"
	"github.com/olekukonko/ll/lx"
)

// fieldBuilderPool pools FieldBuilder instances to reduce allocations.
var fieldBuilderPool = sync.Pool{
	New: func() any {
		return &FieldBuilder{
			fields: make(lx.Fields, 0, 8), // Pre-allocate common size
		}
	},
}

// FieldBuilder enables fluent addition of fields before logging.
// It acts as a builder pattern to attach key-value pairs (fields) to log entries,
// supporting structured logging with metadata. The builder allows chaining to add fields
// and log messages at various levels (Info, Debug, Warn, Error, etc.) in a single expression.
type FieldBuilder struct {
	logger *Logger   // Associated logger instance for logging operations
	fields lx.Fields // Fields to include in the log entry as ordered key-value pairs
}

// getFieldBuilder retrieves a FieldBuilder from the pool or creates a new one.
func getFieldBuilder(logger *Logger, capacity int) *FieldBuilder {
	fb := fieldBuilderPool.Get().(*FieldBuilder)
	fb.logger = logger
	// Ensure minimum capacity to reduce small allocations
	const minFieldCapacity = 4
	if capacity < minFieldCapacity {
		capacity = minFieldCapacity
	}
	if cap(fb.fields) < capacity {
		fb.fields = make(lx.Fields, 0, capacity)
	} else {
		fb.fields = fb.fields[:0] // Reset but keep capacity
	}
	return fb
}

// putFieldBuilder returns a FieldBuilder to the pool for reuse.
func putFieldBuilder(fb *FieldBuilder) {
	fb.logger = nil
	fb.fields = fb.fields[:0]
	fieldBuilderPool.Put(fb)
}

// Logger creates a new logger with the builder's fields embedded in its context.
// It clones the parent logger and copies the builder's fields into the new logger's context,
// enabling persistent field inclusion in subsequent logs. This method supports fluent chaining
// after Fields or Field calls.
// Example:
//
//	logger := New("app").Enable()
//	newLogger := logger.Fields("user", "alice").Logger()
//	newLogger.Info("Action") // Output: [app] INFO: Action [user=alice]
func (fb *FieldBuilder) Logger() *Logger {
	// If logger is nil (e.g., from a false Conditional), return nil
	if fb.logger == nil {
		return nil
	}
	// Clone the parent logger to preserve its configuration
	newLogger := fb.logger.Clone()
	// Copy builder's fields into the new logger's context (optimized)
	if len(fb.fields) > 0 {
		newLogger.context = append(lx.Fields(nil), fb.fields...)
	}
	return newLogger
}

// Info logs a message at Info level with the builder's fields.
// It concatenates the arguments with spaces and delegates to the logger's log method.
// This method is used for informational messages.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Info("Action", "started") // Output: [app] INFO: Action started [user=alice]
func (fb *FieldBuilder) Info(args ...any) {
	if fb.logger == nil {
		return
	}
	fb.logger.log(lx.LevelInfo, lx.ClassText, cat.Space(args...), fb.fields, false)
	putFieldBuilder(fb)
}

// Infof logs a message at Info level with the builder's fields.
// It formats the message using the provided format string and arguments, then delegates
// to the logger's internal log method. This method is part of the fluent API.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Infof("Action %s", "started") // Output: [app] INFO: Action started [user=alice]
func (fb *FieldBuilder) Infof(format string, args ...any) {
	if fb.logger == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fb.logger.log(lx.LevelInfo, lx.ClassText, msg, fb.fields, false)
	putFieldBuilder(fb)
}

// Debug logs a message at Debug level with the builder's fields.
// It concatenates the arguments with spaces and delegates to the logger's log method.
// This method is used for debugging information.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Debug("Debugging", "mode") // Output: [app] DEBUG: Debugging mode [user=alice]
func (fb *FieldBuilder) Debug(args ...any) {
	if fb.logger == nil {
		return
	}
	fb.logger.log(lx.LevelDebug, lx.ClassText, cat.Space(args...), fb.fields, false)
	putFieldBuilder(fb)
}

// Debugf logs a message at Debug level with the builder's fields.
// It formats the message and delegates to the logger's log method.
// This method is used for debugging information that may be disabled in production.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Debugf("Debug %s", "mode") // Output: [app] DEBUG: Debug mode [user=alice]
func (fb *FieldBuilder) Debugf(format string, args ...any) {
	if fb.logger == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fb.logger.log(lx.LevelDebug, lx.ClassText, msg, fb.fields, false)
	putFieldBuilder(fb)
}

// Warn logs a message at Warn level with the builder's fields.
// It concatenates the arguments with spaces and delegates to the logger's log method.
// This method is used for warning conditions.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Warn("Warning", "issued") // Output: [app] WARN: Warning issued [user=alice]
func (fb *FieldBuilder) Warn(args ...any) {
	if fb.logger == nil {
		return
	}
	fb.logger.log(lx.LevelWarn, lx.ClassText, cat.Space(args...), fb.fields, false)
	putFieldBuilder(fb)
}

// Warnf logs a message at Warn level with the builder's fields.
// It formats the message and delegates to the logger's log method.
// This method is used for warning conditions that do not halt execution.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Warnf("Warning %s", "issued") // Output: [app] WARN: Warning issued [user=alice]
func (fb *FieldBuilder) Warnf(format string, args ...any) {
	if fb.logger == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fb.logger.log(lx.LevelWarn, lx.ClassText, msg, fb.fields, false)
	putFieldBuilder(fb)
}

// Error logs a message at Error level with the builder's fields.
// It concatenates the arguments with spaces and delegates to the logger's log method.
// This method is used for error conditions.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Error("Error", "occurred") // Output: [app] ERROR: Error occurred [user=alice]
func (fb *FieldBuilder) Error(args ...any) {
	if fb.logger == nil {
		return
	}
	fb.logger.log(lx.LevelError, lx.ClassText, cat.Space(args...), fb.fields, false)
	putFieldBuilder(fb)
}

// Errorf logs a message at Error level with the builder's fields.
// It formats the message and delegates to the logger's log method.
// This method is used for error conditions that may require attention.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Errorf("Error %s", "occurred") // Output: [app] ERROR: Error occurred [user=alice]
func (fb *FieldBuilder) Errorf(format string, args ...any) {
	if fb.logger == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fb.logger.log(lx.LevelError, lx.ClassText, msg, fb.fields, false)
	putFieldBuilder(fb)
}

// Stack logs a message at Error level with a stack trace and the builder's fields.
// It concatenates the arguments with spaces and delegates to the logger's log method.
// This method is useful for debugging critical errors.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Stack("Critical", "error") // Output: [app] ERROR: Critical error [user=alice stack=...]
func (fb *FieldBuilder) Stack(args ...any) {
	if fb.logger == nil {
		return
	}
	fb.logger.log(lx.LevelError, lx.ClassText, cat.Space(args...), fb.fields, true)
	putFieldBuilder(fb)
}

// Stackf logs a message at Error level with a stack trace and the builder's fields.
// It formats the message and delegates to the logger's log method.
// This method is useful for debugging critical errors.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Stackf("Critical %s", "error") // Output: [app] ERROR: Critical error [user=alice stack=...]
func (fb *FieldBuilder) Stackf(format string, args ...any) {
	if fb.logger == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fb.logger.log(lx.LevelError, lx.ClassText, msg, fb.fields, true)
	putFieldBuilder(fb)
}

// Fatal logs a message at Error level with a stack trace and the builder's fields, then exits.
// It constructs the message from variadic arguments, logs it with a stack trace, and terminates
// the program with exit code 1. This method is used for unrecoverable errors.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Fatal("Fatal", "error") // Output: [app] ERROR: Fatal error [user=alice stack=...], then exits
func (fb *FieldBuilder) Fatal(args ...any) {
	if fb.logger == nil {
		return
	}
	var builder strings.Builder
	for i, arg := range args {
		if i > 0 {
			builder.WriteString(lx.Space)
		}
		builder.WriteString(fmt.Sprint(arg))
	}
	fb.logger.log(lx.LevelFatal, lx.ClassText, builder.String(), fb.fields, fb.logger.fatalStack)
	if fb.logger.fatalExits {
		os.Exit(1)
	}
	putFieldBuilder(fb)
}

// Fatalf logs a formatted message at Error level with a stack trace and the builder's fields,
// then exits. It delegates to Fatal. This method is used for unrecoverable errors.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Fatalf("Fatal %s", "error") // Output: [app] ERROR: Fatal error [user=alice stack=...], then exits
func (fb *FieldBuilder) Fatalf(format string, args ...any) {
	if fb.logger == nil {
		return
	}
	fb.Fatal(fmt.Sprintf(format, args...))
}

// Panic logs a message at Error level with a stack trace and the builder's fields, then panics.
// It constructs the message from variadic arguments, logs it with a stack trace, and triggers
// a panic with the message. This method is used for critical errors.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Panic("Panic", "error") // Output: [app] ERROR: Panic error [user=alice stack=...], then panics
func (fb *FieldBuilder) Panic(args ...any) {
	if fb.logger == nil {
		return
	}
	var builder strings.Builder
	for i, arg := range args {
		if i > 0 {
			builder.WriteString(lx.Space)
		}
		builder.WriteString(fmt.Sprint(arg))
	}
	msg := builder.String()
	fb.logger.log(lx.LevelError, lx.ClassText, msg, fb.fields, true)
	panic(msg)
}

// Panicf logs a formatted message at Error level with a stack trace and the builder's fields,
// then panics. It delegates to Panic. This method is used for critical errors.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Panicf("Panic %s", "error") // Output: [app] ERROR: Panic error [user=alice stack=...], then panics
func (fb *FieldBuilder) Panicf(format string, args ...any) {
	if fb.logger == nil {
		return
	}
	fb.Panic(fmt.Sprintf(format, args...))
}

// Err adds one or more errors to the FieldBuilder as a field and logs them.
// It stores non-nil errors in the "error" field: a single error if only one is non-nil,
// or a slice of errors if multiple are non-nil. Returns the FieldBuilder for chaining.
// Example:
//
//	logger := New("app").Enable()
//	err1 := errors.New("failed 1")
//	err2 := errors.New("failed 2")
//	logger.Fields("k", "v").Err(err1, err2).Info("Error occurred")
func (fb *FieldBuilder) Err(errs ...error) *FieldBuilder {
	if fb.logger == nil {
		return fb
	}
	var nonNilErrors []error
	var builder strings.Builder
	count := 0
	for i, err := range errs {
		if err != nil {
			if i > 0 && count > 0 {
				builder.WriteString("; ")
			}
			builder.WriteString(err.Error())
			nonNilErrors = append(nonNilErrors, err)
			count++
		}
	}
	if count > 0 {
		if count == 1 {
			fb.fields = append(fb.fields, lx.Field{Key: "error", Value: nonNilErrors[0]})
		} else {
			fb.fields = append(fb.fields, lx.Field{Key: "error", Value: nonNilErrors})
		}
		fb.logger.log(lx.LevelError, lx.ClassText, builder.String(), nil, false)
	}
	return fb
}

// Merge adds additional key-value pairs to the FieldBuilder.
// It processes variadic arguments as key-value pairs, expecting string keys.
// Returns the FieldBuilder for chaining.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("k1", "v1").Merge("k2", "v2").Info("Action") // Output: [app] INFO: Action [k1=v1 k2=v2]
func (fb *FieldBuilder) Merge(pairs ...any) *FieldBuilder {
	// Merge can work even with nil logger since it just manipulates fields
	for i := 0; i < len(pairs)-1; i += 2 {
		if key, ok := pairs[i].(string); ok {
			fb.fields = append(fb.fields, lx.Field{Key: key, Value: pairs[i+1]})
		} else {
			fb.fields = append(fb.fields, lx.Field{
				Key:   "error",
				Value: fmt.Errorf("non-string key in Merge: %v", pairs[i]),
			})
		}
	}
	if len(pairs)%2 != 0 {
		fb.fields = append(fb.fields, lx.Field{
			Key:   "error",
			Value: fmt.Errorf("uneven key-value pairs in Merge: [%v]", pairs[len(pairs)-1]),
		})
	}
	return fb
}
