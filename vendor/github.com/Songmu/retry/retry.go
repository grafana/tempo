package retry

import (
	"context"
	"time"
)

// Retry calls the `fn` and if it returns the error, retry to call `fn` after `interval` duration.
// The `fn` is called up to `n` times.
func Retry(n uint, interval time.Duration, fn func() error) (err error) {
	return WithContext(context.Background(), n, interval, fn)
}

// WithContext stops retrying when the context is done.
func WithContext(ctx context.Context, n uint, interval time.Duration, fn func() error) (err error) {
	for n > 0 {
		n--
		err = fn()
		if err == nil || n <= 0 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
	return
}
