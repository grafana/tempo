package test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kit/log"
)

var _ log.Logger = (*TestingLogger)(nil)

type TestingLogger struct {
	t    testing.TB
	mtx  *sync.Mutex
	done atomic.Bool
}

func NewTestingLogger(t testing.TB) *TestingLogger {
	logger := &TestingLogger{
		t:   t,
		mtx: &sync.Mutex{},
	}
	registerCleanup(t, logger)
	return logger
}

// WithT returns a new logger that logs to t. Writes between the new logger and the original logger are synchronized.
func (l *TestingLogger) WithT(t testing.TB) log.Logger {
	child := &TestingLogger{
		t:   t,
		mtx: l.mtx,
	}
	registerCleanup(t, child)
	return child
}

func (l *TestingLogger) Log(keyvals ...interface{}) error {
	if l.done.Load() {
		return nil
	}

	// Prepend log with timestamp.
	keyvals = append([]interface{}{time.Now().String()}, keyvals...)

	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.done.Load() {
		return nil
	}

	l.t.Log(keyvals...)

	return nil
}

func registerCleanup(t testing.TB, l *TestingLogger) {
	t.Cleanup(func() {
		l.done.Store(true)
	})
}
