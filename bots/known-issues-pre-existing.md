# Known Pre-existing Test Failures

These test failures existed BEFORE the Phase 1 v2 critical fixes and are NOT regressions from our implementation.

## TestInstanceSearchDoesNotRace

**Status:** Pre-existing CRITICAL issue
**Severity:** HIGH
**Phase:** Defer to Phase 2 or Phase 3

**Issue:**
File I/O race condition during concurrent search operations. Files being deleted while searches are reading them.

**Error:**
```
unexpected error searching tag values: error opening file .../0000000001:
no such file or directory
```

**Root Cause:**
Missing synchronization between block lifecycle and search operations. Block files can be deleted while concurrent search operations are accessing them.

**Recommendation:**
- Add reference counting for block files
- Implement proper synchronization between search and deletion
- Add file existence checks before opening
- Consider file locking during search operations

**Deferred Reason:**
This requires architectural changes to block lifecycle management and should be addressed in a dedicated phase after Phase 1 v2 critical fixes are validated.

---

## TestConsume_GracefulShutdown_ReturnsNilError

**Status:** Pre-existing HIGH issue (HIGH-6 from original review)
**Severity:** HIGH
**Phase:** Defer to Phase 2

**Issue:**
Kafka consumer enters Failed state instead of Terminated during graceful shutdown. Context cancelled before offset commit completes.

**Error:**
```
invalid service state: Failed, expected: Terminated
failure: failed to fetch last committed offset: context canceled
```

**Root Cause:**
Context cancellation happens before Kafka offset commit operations complete, causing reader to enter Failed state.

**Recommendation:**
- Handle context.Canceled errors gracefully in Kafka consumer
- Separate critical shutdown operations from cancellable context
- Use timeout instead of immediate cancellation
- Ensure offset commits complete before context cancellation

**Deferred Reason:**
This requires changes to Kafka consumer integration and graceful shutdown coordination. Should be addressed after Phase 1 v2 critical fixes are validated.

---

## Decision

Both issues are real problems that need fixing, but:
1. They are NOT regressions from Phase 1 v2 critical fixes
2. They require architectural changes beyond the scope of critical fixes
3. Fixing them now would delay validation of the 5 CRITICAL fixes
4. They should be addressed in dedicated phases (Phase 2 or later)

**Recommendation:** Document these as known issues and proceed with REVIEW phase for Phase 1 v2 critical fixes. Address these issues in subsequent phases.
