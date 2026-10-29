package backoff

import (
	"errors"
	"fmt"
	"time"
)

// Cause values reported by RetryError.Cause. Match them with errors.Is.
var (
	// ErrPermanent is the cause when the operation returned a Permanent error.
	ErrPermanent = errors.New("backoff: permanent error")

	// ErrExhausted is the cause when retrying stops because WithMaxTries was
	// reached or the backoff policy returned Stop.
	ErrExhausted = errors.New("backoff: retries exhausted")

	// ErrMaxElapsedTime is the cause when retrying stops because
	// WithMaxElapsedTime was reached.
	ErrMaxElapsedTime = errors.New("backoff: maximum elapsed time exceeded")
)

// RetryError is the error returned by Retry for every failure. It records the
// last error returned by the operation (LastErr) together with the reason
// retrying stopped (Cause), so callers never lose either piece of information.
//
// Inspect it with errors.Is, errors.As, or AsRetryError:
//
//	result, err := backoff.Retry(ctx, op)
//	switch {
//	case errors.Is(err, backoff.ErrPermanent):
//		// operation returned a Permanent error
//	case errors.Is(err, context.Canceled):
//		// caller cancelled ctx
//	case errors.Is(err, backoff.ErrMaxElapsedTime):
//		// ran out of the WithMaxElapsedTime budget
//	case errors.Is(err, backoff.ErrExhausted):
//		// hit WithMaxTries or the backoff policy stopped
//	}
//
//	if re := backoff.AsRetryError(err); re != nil {
//		log.Printf("gave up after last error: %v", re.LastErr)
//	}
//
// Because RetryError implements Unwrap() []error, errors.Unwrap (the single
// error form) returns nil for it; use errors.Is, errors.As, or AsRetryError.
type RetryError struct {
	// LastErr is the error returned by the final operation attempt. For a
	// permanent failure it is the error passed to Permanent.
	LastErr error
	// Cause reports why retrying stopped: ErrPermanent, ErrExhausted,
	// ErrMaxElapsedTime, or a context cancellation cause (see context.Cause).
	Cause error
}

// Error returns a single-line representation of the cause and last error.
func (e *RetryError) Error() string {
	return fmt.Sprintf("%s (last error: %s)", e.Cause, e.LastErr)
}

// Unwrap returns the cause and the last operation error so both can be
// matched with errors.Is and errors.As.
func (e *RetryError) Unwrap() []error {
	return []error{e.Cause, e.LastErr}
}

// AsRetryError returns the *RetryError in err's chain, or nil if there is none
// (including when err is nil). It is a convenience wrapper around errors.As.
func AsRetryError(err error) *RetryError {
	var re *RetryError
	errors.As(err, &re)
	return re
}

// permanent marks an operation error as non-retriable. It is an internal
// transport produced by Permanent and consumed by Retry, which converts it
// into a RetryError with Cause ErrPermanent. It is never returned to callers.
type permanent struct {
	err error
}

// Permanent wraps err to signal that Retry should stop immediately instead of
// retrying. Retry then returns a *RetryError with Cause ErrPermanent and
// LastErr set to err. Permanent(nil) returns nil.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanent{err: err}
}

// Error returns the wrapped error's message.
func (e *permanent) Error() string { return e.err.Error() }

// Unwrap returns the wrapped error.
func (e *permanent) Unwrap() error { return e.err }

// Is reports a match against ErrPermanent so a Permanent error can be detected
// with errors.Is even before Retry converts it into a RetryError.
func (e *permanent) Is(target error) bool { return target == ErrPermanent }

// RetryAfterError signals that the operation should be retried after the given
// duration. When an operation returns one (directly or wrapped), Retry waits
// that duration before the next attempt and resets the backoff policy, so the
// backoff sequence restarts afterward.
//
// The error that triggered the wait (passed to RetryAfter) is available via
// Unwrap, so errors.Is and errors.As see through the RetryAfterError. If
// retrying later stops because a limit is reached or the context ends, Retry
// reports that error as RetryError.LastErr instead of the RetryAfterError
// itself, so the underlying cause is not lost.
type RetryAfterError struct {
	Duration time.Duration
	err      error
}

// RetryAfter returns a RetryAfterError that tells Retry to wait the given
// duration before the next attempt. cause is the error that triggered the
// wait; it is preserved as RetryError.LastErr if retrying stops. Pass a non-nil
// cause so the failure reason is not lost; nil is allowed but discouraged.
func RetryAfter(d time.Duration, cause error) error {
	return &RetryAfterError{
		Duration: d,
		err:      cause,
	}
}

// Error returns a string representation of the RetryAfter error.
func (e *RetryAfterError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s (retry after %s)", e.err, e.Duration)
	}
	return fmt.Sprintf("retry after %s", e.Duration)
}

// Unwrap returns the error that triggered the retry, if one was provided.
func (e *RetryAfterError) Unwrap() error { return e.err }
