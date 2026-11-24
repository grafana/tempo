package pipeline

import (
	"context"
	"sync/atomic"
)

type dequeuedCtxKey struct{}

// WithDequeueMarker returns a context for a job with a marker that the queue sets once a querier takes the
// job, see MarkDequeued. The returned bool reports whether that happened.
func WithDequeueMarker(ctx context.Context) (context.Context, *atomic.Bool) {
	dequeued := &atomic.Bool{}
	return context.WithValue(ctx, dequeuedCtxKey{}, dequeued), dequeued
}

// MarkDequeued marks the job of ctx as taken by a querier. Without a marker on ctx it does nothing.
func MarkDequeued(ctx context.Context) {
	if dequeued, ok := ctx.Value(dequeuedCtxKey{}).(*atomic.Bool); ok {
		dequeued.Store(true)
	}
}
