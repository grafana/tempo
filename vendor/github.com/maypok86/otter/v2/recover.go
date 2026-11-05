// Copyright (c) 2025 Alexey Mayshev and contributors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otter

import (
	"context"
	"time"

	"github.com/maypok86/otter/v2/stats"
)

// recoverCallback recovers a panic of a user callback that only observes the cache (a deletion
// listener, a stats recorder, an executor running one) and logs it. Such a callback often runs
// with the cache's locks held or in the middle of maintenance, where a panic would leave the
// cache locked or half-updated, so its failure must not propagate.
//
// It must be deferred directly.
func recoverCallback(logger Logger, msg string) {
	if r := recover(); r != nil {
		logger.Error(context.Background(), msg, newPanicError(r))
	}
}

// safeRecorder is a stats.Recorder that recovers and logs panics of the user's recorder. The
// cache records stats between committing a change and replaying it to the policies, where a
// panic would leave the change out of the policies. A runtime.Goexit cannot be stopped, so a
// write is handed to the policies in a defer around the recorder (afterWrite) or before it runs
// (doCompute); a removal counts its eviction first and leaves a lost task to the timer wheel
// (publish, deleteNodeFromMap).
type safeRecorder struct {
	recorder stats.Recorder
	logger   Logger
}

func (r *safeRecorder) RecordHits(count int) {
	defer recoverCallback(r.logger, "StatsRecorder.RecordHits panicked")
	r.recorder.RecordHits(count)
}

func (r *safeRecorder) RecordMisses(count int) {
	defer recoverCallback(r.logger, "StatsRecorder.RecordMisses panicked")
	r.recorder.RecordMisses(count)
}

func (r *safeRecorder) RecordEviction(weight uint32) {
	defer recoverCallback(r.logger, "StatsRecorder.RecordEviction panicked")
	r.recorder.RecordEviction(weight)
}

func (r *safeRecorder) RecordLoadSuccess(loadTime time.Duration) {
	defer recoverCallback(r.logger, "StatsRecorder.RecordLoadSuccess panicked")
	r.recorder.RecordLoadSuccess(loadTime)
}

func (r *safeRecorder) RecordLoadFailure(loadTime time.Duration) {
	defer recoverCallback(r.logger, "StatsRecorder.RecordLoadFailure panicked")
	r.recorder.RecordLoadFailure(loadTime)
}

// safeLogger is a Logger that drops panics of the user's logger. The cache logs from its recovery
// paths and from refresh goroutines, where a panicking logger would undo the recovery (leave the
// eviction lock held or a node half removed) or crash the process.
type safeLogger struct {
	logger Logger
}

func (l *safeLogger) Warn(ctx context.Context, msg string, err error) {
	//nolint:errcheck // the logger's panic is dropped: there is nowhere left to report it
	defer func() { _ = recover() }()
	l.logger.Warn(ctx, msg, err)
}

func (l *safeLogger) Error(ctx context.Context, msg string, err error) {
	//nolint:errcheck // the logger's panic is dropped: there is nowhere left to report it
	defer func() { _ = recover() }()
	l.logger.Error(ctx, msg, err)
}
