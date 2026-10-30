package backoff

import (
	"context"
	"errors"
	"time"
)

// DefaultMaxElapsedTime sets a default limit for the total retry duration.
const DefaultMaxElapsedTime = 15 * time.Minute

// Operation is the function Retry calls. It is invoked at least once and may be
// retried on error. Return a Permanent error to stop retrying immediately, or a
// RetryAfterError to control the delay before the next attempt.
type Operation[T any] func() (T, error)

// Notify is called after a failed attempt that will be retried, with the
// operation error and the backoff duration before the next attempt. It is
// called once per retry, not for the final error that stops Retry (a permanent
// error, an exhausted limit, or a cancelled context).
type Notify func(error, time.Duration)

// retryOptions holds configuration settings for the retry mechanism.
type retryOptions struct {
	BackOff        BackOff       // Strategy for calculating backoff periods.
	Timer          timer         // Timer to manage retry delays.
	Notify         Notify        // Optional function called before each backoff wait.
	MaxTries       uint          // Maximum number of retry attempts.
	MaxElapsedTime time.Duration // Maximum total time for all retries.
}

// RetryOption configures the behavior of Retry.
type RetryOption func(*retryOptions)

// WithBackOff configures the backoff policy used between attempts. The default
// is NewExponentialBackOff.
//
// Retry calls Reset on the policy before the first attempt, so a previously
// used policy may be passed. A BackOff is stateful and not safe for concurrent
// use: give each concurrent Retry call its own BackOff rather than sharing one.
func WithBackOff(b BackOff) RetryOption {
	return func(args *retryOptions) {
		args.BackOff = b
	}
}

// withTimer sets a custom timer for managing delays between retries.
func withTimer(t timer) RetryOption {
	return func(args *retryOptions) {
		args.Timer = t
	}
}

// WithNotify sets a function called after each failed attempt that will be
// retried. See Notify for exactly when it fires.
func WithNotify(n Notify) RetryOption {
	return func(args *retryOptions) {
		args.Notify = n
	}
}

// WithMaxTries limits the total number of attempts, not retries: WithMaxTries(1)
// runs the operation once and does not retry. When the limit is reached, Retry
// returns a *RetryError with Cause ErrExhausted. The default, 0, means no limit.
func WithMaxTries(n uint) RetryOption {
	return func(args *retryOptions) {
		args.MaxTries = n
	}
}

// WithMaxElapsedTime limits the total wall-clock time spent retrying, measured
// from when Retry is called. When the limit is reached, Retry returns a
// *RetryError with Cause ErrMaxElapsedTime.
//
// The limit is checked only between attempts: it gates whether another attempt
// is scheduled. It does not interrupt an operation that is already running, nor
// a backoff wait already in progress, and Retry stops early rather than
// starting a backoff that would overrun the limit.
//
// This differs from bounding Retry with a context deadline (e.g.
// context.WithTimeout): a context deadline is reactive — it interrupts the
// backoff wait and, if the operation observes the context, can abort an
// in-flight attempt — and Retry reports it with Cause context.DeadlineExceeded.
//
// The default is DefaultMaxElapsedTime (15 minutes), so both limits are active
// at once unless overridden. Pass 0 to disable the elapsed-time limit and rely
// on the context (or WithMaxTries) instead.
func WithMaxElapsedTime(d time.Duration) RetryOption {
	return func(args *retryOptions) {
		args.MaxElapsedTime = d
	}
}

// Retry attempts the operation until it succeeds, returns a Permanent error,
// or backoff completes. It ensures the operation is executed at least once.
//
// On success it returns the operation result and a nil error. On any failure
// it returns the last result and a *RetryError whose Cause reports why it
// stopped — ErrPermanent, ErrExhausted, ErrMaxElapsedTime, or the context
// cancellation cause — and whose LastErr holds the last operation error.
// See RetryError and AsRetryError.
//
// ctx bounds the retry loop: its cancellation or deadline stops further
// attempts and interrupts the wait between them. The operation receives no
// context, so capture ctx inside the operation if you want cancellation to
// abort an in-flight attempt. To bound only how long backoff keeps retrying,
// without affecting in-flight attempts, use WithMaxElapsedTime instead.
func Retry[T any](ctx context.Context, operation Operation[T], opts ...RetryOption) (T, error) {
	// Initialize default retry options.
	args := &retryOptions{
		BackOff:        NewExponentialBackOff(),
		Timer:          &defaultTimer{},
		MaxElapsedTime: DefaultMaxElapsedTime,
	}

	// Apply user-provided options to the default settings.
	for _, opt := range opts {
		opt(args)
	}

	defer args.Timer.Stop()

	startedAt := time.Now()
	args.BackOff.Reset()
	for numTries := uint(1); ; numTries++ {
		// Execute the operation.
		res, err := operation()
		if err == nil {
			return res, nil
		}

		// Stop immediately on a permanent error; surface it as a RetryError.
		var perm *permanent
		if errors.As(err, &perm) {
			return res, &RetryError{LastErr: perm.err, Cause: ErrPermanent}
		}

		// A RetryAfterError carries the delay before the next attempt; if it also
		// carries an underlying error, that is the meaningful error to report as
		// LastErr should retrying stop (mirrors how a permanent error surfaces its
		// inner error). errors.As matches it whether returned directly or wrapped.
		lastErr := err
		var retryAfter *RetryAfterError
		if errors.As(err, &retryAfter) && retryAfter.err != nil {
			lastErr = retryAfter.err
		}

		// Stop retrying if maximum tries exceeded.
		if args.MaxTries > 0 && numTries >= args.MaxTries {
			return res, &RetryError{LastErr: lastErr, Cause: ErrExhausted}
		}

		// Stop retrying if context is cancelled.
		if cerr := context.Cause(ctx); cerr != nil {
			return res, &RetryError{LastErr: lastErr, Cause: cerr}
		}

		// Calculate next backoff duration.
		next := args.BackOff.NextBackOff()
		if next == Stop {
			return res, &RetryError{LastErr: lastErr, Cause: ErrExhausted}
		}

		// Reset backoff if a RetryAfterError requested a specific delay.
		if retryAfter != nil {
			next = retryAfter.Duration
			args.BackOff.Reset()
		}

		// Stop retrying if maximum elapsed time exceeded.
		if args.MaxElapsedTime > 0 && time.Since(startedAt)+next > args.MaxElapsedTime {
			return res, &RetryError{LastErr: lastErr, Cause: ErrMaxElapsedTime}
		}

		// Notify on error if a notifier function is provided.
		if args.Notify != nil {
			args.Notify(err, next)
		}

		// Wait for the next backoff period or context cancellation.
		args.Timer.Start(next)
		select {
		case <-args.Timer.C():
		case <-ctx.Done():
			return res, &RetryError{LastErr: lastErr, Cause: context.Cause(ctx)}
		}
	}
}
