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
	"math"
	"time"

	"github.com/maypok86/otter/v2/internal/xmath"
)

// Deadlines are an entry's expiration and refresh times. The functions in this file decide them
// from what happened to the entry, the calculators and the current time. They never touch a
// node: the cache decides how to store the result.

// expireTolerance is how far a read may move an entry's deadline without writing it. The clock
// counts nanoseconds, so with an expiration that is reset on every read, each read would store
// a new deadline, and readers of a hot key would all write to the same cache line. The deadline
// is left as it is only while both the current and the new lifetime are longer than
// expireTolerance, so an entry may expire up to expireTolerance early, or late if the
// calculator shortens its lifetime by less than that.
const expireTolerance = time.Second

// writeKind is what a write does to an entry, which decides the calculator method that sets
// its deadlines.
type writeKind uint8

const (
	// createWrite writes a key that is absent or whose entry has expired. The entry it shows
	// the calculators has no deadlines.
	createWrite writeKind = iota
	// updateWrite replaces a live value.
	updateWrite
	// reloadWrite replaces a live value with the result of its refresh.
	reloadWrite
	// reloadFailure keeps a live value whose refresh failed; only the refresh time changes.
	reloadFailure
)

// newWriteKind returns the kind of a write over an entry that is live or not, and that comes
// from a refresh or not. A failed refresh does not write, so it is never returned.
func newWriteKind(oldLive, isRefresh bool) writeKind {
	switch {
	case !oldLive:
		return createWrite
	case isRefresh:
		return reloadWrite
	default:
		return updateWrite
	}
}

// expiresAtAfterWrite returns the expiration time of entry after a write of the given kind.
// The expiry calculator does not tell a refresh from an update.
func expiresAtAfterWrite[K comparable, V any](
	calc ExpiryCalculator[K, V],
	kind writeKind,
	entry Entry[K, V],
	oldValue V,
	nowNano int64,
) int64 {
	var expiresAfter time.Duration
	if kind == createWrite {
		expiresAfter = calc.ExpireAfterCreate(entry)
	} else {
		expiresAfter = calc.ExpireAfterUpdate(entry, oldValue)
	}
	return nextDeadline(entry.ExpiresAtNano, nowNano, expiresAfter)
}

// refreshableAtAfterWrite returns the refresh time of entry after a write of the given kind.
// err is the error of a failed refresh (reloadFailure).
func refreshableAtAfterWrite[K comparable, V any](
	calc RefreshCalculator[K, V],
	kind writeKind,
	entry Entry[K, V],
	oldValue V,
	err error,
	nowNano int64,
) int64 {
	var refreshableAfter time.Duration
	switch kind {
	case createWrite:
		refreshableAfter = calc.RefreshAfterCreate(entry)
	case updateWrite:
		refreshableAfter = calc.RefreshAfterUpdate(entry, oldValue)
	case reloadWrite:
		refreshableAfter = calc.RefreshAfterReload(entry, oldValue)
	case reloadFailure:
		refreshableAfter = calc.RefreshAfterReloadFailure(entry, err)
	}
	return nextDeadline(entry.RefreshableAtNano, nowNano, refreshableAfter)
}

// expiresAtAfterRead returns the expiration time that a read of entry moves it to, and whether
// it moves it at all (see expireTolerance).
func expiresAtAfterRead[K comparable, V any](calc ExpiryCalculator[K, V], entry Entry[K, V], nowNano int64) (int64, bool) {
	return moveDeadline(entry.ExpiresAtNano, nowNano, calc.ExpireAfterRead(entry), expireTolerance)
}

// nextDeadline returns nowNano + d when the calculator returned a positive d that differs from
// the time left until current, and current otherwise. A calculator returns the time left to keep
// the deadline, and a non-positive duration keeps it too.
func nextDeadline(current, nowNano int64, d time.Duration) int64 {
	if d > 0 && time.Duration(current-nowNano) != d {
		return deadlineAfter(nowNano, d)
	}
	return current
}

// moveDeadline returns the deadline nowNano + d that replaces current, and whether it replaces
// it. While both the current and the new lifetime are longer than tolerance, a deadline that
// would move by at most tolerance is left as it is.
func moveDeadline(current, nowNano int64, d, tolerance time.Duration) (int64, bool) {
	if d <= 0 {
		return current, false
	}

	next := deadlineAfter(nowNano, d)
	if next == current {
		// Also a deadline that is never reached: storing it again would still take the cache line.
		return current, false
	}
	currentDuration := time.Duration(current - nowNano)
	diff := time.Duration(xmath.Abs(int64(d - currentDuration)))
	if diff <= tolerance && d > tolerance && currentDuration > tolerance {
		return current, false
	}
	return next, true
}

// deadlineAfter returns nowNano + d, saturated at math.MaxInt64, the time that is never reached.
// The documented "never" duration is math.MaxInt64, and with a clock that counts from the Unix
// epoch the plain sum would wrap to a deadline in the past. A sum with a negative nowNano cannot
// overflow.
func deadlineAfter(nowNano int64, d time.Duration) int64 {
	if nowNano > 0 && int64(d) > math.MaxInt64-nowNano {
		return math.MaxInt64
	}
	return nowNano + int64(d)
}
