# Code Review Standards

Multi-domain review framework for Go code changes. Work through each pass systematically. For every issue found, record: what, why, where (`file:line`), severity, and a concrete fix.

---

## Severity Levels

**CRITICAL** — Exploitable vulnerabilities, data loss, crashes
- SQL/command injection, authentication bypass, nil dereference in hot path, hardcoded credentials

**HIGH** — Serious bugs, breaking behavior, security weaknesses
- Missing error handling in critical paths, race conditions, resource leaks, goroutine leaks, weak cryptography, breaking API changes

**MEDIUM** — Quality issues, potential bugs, non-idiomatic code
- Missing validation, non-idiomatic Go, N+1 queries, missing caching, minor reference issues

**LOW** — Style, minor improvements, docs
- Comment typos, naming suggestions, missing doc comments on non-critical exports

**Routing:**
- Any CRITICAL or HIGH → must fix before merging; consider re-examining the design
- MEDIUM/LOW only → targeted fixes, non-blocking

---

## Pass 1: Security

Focus: OWASP Top 10, secrets, auth/authz, cryptography, input validation.

- Injection vulnerabilities (command, path traversal)
- Hardcoded credentials or API keys
- **Config fields that hold secrets typed as `string` instead of `flagext.Secret` or `config.Secret` (`github.com/prometheus/common/config`)** — any field whose name suggests a password, token, key, or credential must use a secret type so it is redacted in logs and marshalled output. Check new and changed config structs for fields named `password`, `token`, `key`, `secret`, `credential`, `auth`, or similar.
- Missing authentication or authorization checks
- Weak/missing cryptography (`crypto/md5`, `crypto/sha1`, `math/rand` for security purposes)
- Missing input validation on external data
- XSS, CSRF, unsafe HTML rendering
- JWT issues (no expiry, weak signing)

```bash
# Command execution
grep -rn "os/exec\|exec.CommandContext\|exec.Command\|syscall.Exec" . --include="*.go"
# Potential secrets in code
grep -rEn "api_key|apikey|password|secret|private_key" . --include="*.go" --include="*.yaml"
# Config fields that may need flagext.Secret or config.Secret instead of string
grep -rEn "(Password|Token|Key|Secret|Credential|Auth)\s+string" . --include="*.go"
# Weak crypto
grep -rn "crypto/md5\|crypto/sha1\|math/rand" . --include="*.go"
```

---

## Pass 2: Bug Diagnosis

Focus: Nil pointer dereferences, race conditions, off-by-one errors, resource leaks.

- Nil pointer dereferences (dereferencing without nil check)
- Goroutine/data race conditions (shared state without locks)
- Off-by-one errors in loops and slices
- Unclosed resources (files, connections, response bodies)
- Infinite loops or missing termination conditions
- Incorrect type assertions without `ok` check
- Shadowed variables causing logic bugs

```bash
# Unchecked type assertion candidates — verify each has an ok check or is intentional
grep -rEn "\\.\\([^)]+\\)" . --include="*.go"
```

---

## Pass 3: Error Handling

Focus: Error handling consistency, silent failures, missing checks.

- Ignored errors (`err` assigned but never checked)
- Silent error swallowing (`_ = someFunc()`)
- Missing error wrapping context — use `fmt.Errorf("...: %w", err)`
- Errors logged but not returned, or returned but not logged
- Missing timeout handling on external calls
- **Panic used instead of error return** — never panic; use error returns for all failures

```bash
# Errors silenced
grep -rn "_ = \|_, _ =" . --include="*.go"
# err assigned — scan for unchecked assignments
grep -rn "err :=" . --include="*.go"
```

---

## Pass 4: Code Quality & Logic

Focus: Bugs, logic errors, missing edge cases, correctness.

- Logic errors and incorrect conditional branches
- Missing edge case handling (empty input, nil, zero values)
- Incorrect use of library APIs
- Cross-file consistency (config field names, function signatures, enum values)
- Dead code or unreachable branches
- Test coverage gaps on critical paths

---

## Pass 5: Performance

Focus: Algorithmic complexity, allocation patterns, hot path costs.

- O(n²) or worse algorithms where O(n log n) is achievable
- Allocations inside tight loops (check with `-benchmem`)
- Missing caching on repeated expensive calls
- String concatenation in loops — use `strings.Builder`
- Large value types copied instead of passed by pointer
- Goroutine leaks
- Parquet iterators scanning exhausted row groups instead of using `SeekTo()`
- **Performance regressions on a known hot path must be justified with benchmarks** — known hot paths are: `instance.push()`, `liveTrace.Push()`, TraceQL `BinaryOperation.execute()`, TraceQL group evaluation, Parquet row number comparison, Parquet block TraceQL execution. A slower implementation is acceptable if the path is not performance-critical, but that must be stated explicitly
- Microbenchmarks (`go test -bench`) are valuable and encouraged, especially for isolated paths like TraceQL engine methods — less CPU and less memory is an improvement regardless of scale. When a change has an unclear trade-off (e.g., less CPU but more memory), a production profile of the affected area helps determine whether the trade-off is acceptable

```bash
# String concatenation in loops
grep -rn "+=.*\"" . --include="*.go"
# Benchmarks with allocation counts
go test -bench=. -benchmem ./...
# Escape analysis — check what moves to heap
go build -gcflags="-m" ./...
```

---

## Pass 6: Go-Specific Idioms

Focus: Idiomatic Go, concurrency patterns, naming conventions.

- Non-idiomatic naming (stuttered names like `pkg.PkgType`, snake_case in Go identifiers)
- Interface misuse (too large, defined in wrong package)
- Goroutines without `WaitGroup` or context cancellation
- Missing context propagation (`ctx` parameter not threaded through)
- Error comparison with `==` instead of `errors.Is`/`errors.As`
- `defer` inside loops
- `init()` overuse

---

## Pass 7: Architecture & Design

Focus: Design patterns, separation of concerns, testability.

- Tight coupling between packages or layers
- Missing abstraction where it would reduce meaningful duplication
- Single responsibility principle violations
- Circular dependencies
- Global state that hinders testability
- Overly complex function signatures

---

## Pass 8: Documentation

Focus: README accuracy, comment correctness, example validity, API doc alignment.

- Examples that don't compile or don't match current API
- Missing or incorrect doc comments on exported symbols
- README commands or configs that no longer work
- Comments describing removed functionality

---

## Pass 9: Comment Accuracy

Focus: Are inline comments truthful? A comment that lies is worse than no comment.

- **Stale comments** — code changed but comment wasn't updated
- **Misleading comments** — comment says X but code does Y
- **Wrong parameter/return descriptions** — e.g., says seconds but code uses milliseconds
- **Unresolved TODO/FIXME** that should have been addressed in this changeset
- **Commented-out code** with no explanation

```bash
# Find TODO/FIXME in changed files
git diff HEAD --name-only | xargs grep -n "TODO\|FIXME\|HACK\|XXX" 2>/dev/null
```

Severity: misleading/stale that could cause a bug → HIGH; contradicts code but low-harm → MEDIUM; unresolved TODO for this change → MEDIUM; commented-out code → LOW.

---

## Pass 10: Reference Integrity

Focus: Every reference in a comment to a spec file or design doc must point to something real.

For each referenced file:
1. Verify the file exists
2. Verify the referenced section/invariant exists within it
3. Verify the code actually matches what the referenced section says

Severity: referenced file doesn't exist → HIGH; referenced section doesn't exist → HIGH; file exists but code contradicts it → HIGH; vague/unverifiable reference → LOW.

---

## Output Format

```markdown
## Critical Issues (Must Fix)

### Issue 1: [Title]
**Severity:** CRITICAL
**Domain:** security
**File:** auth/login.go:45
**Description:** [What the problem is]
**Impact:** [What could happen]
**Fix:** [Concrete suggestion]

## High Priority Issues
...

## Medium Priority Issues
...

## Low Priority Issues
...

## Summary
- CRITICAL: N | HIGH: N | MEDIUM: N | LOW: N
- Recommendation: MUST FIX / TARGETED FIXES / READY
```

---

## Tempo Maintainer Standards

Patterns observed from code reviews by core maintainers. These reflect what gets caught in practice.

### Test Specificity

Tests must lock in full behavioral contracts, not spot-check with loose assertions.

```go
// Flagged: loose check
assert.Contains(t, result, "<_>")

// Preferred: full behavioral assertion
assert.Equal(t, "GET /api/users/<_>", result)
```

Cover edge cases explicitly: multi-byte characters, empty strings, special characters, nil inputs. Name subtests to describe what they test, not just the input.

### Concurrency Assumptions Must Be Explicit

When a slice or pointer is shared across goroutines, document whether it's safe or must be copied. Don't assume callers know.

```go
// Add comment explaining why a copy is made
flushes := make([]*flush, len(s.flushes))
copy(flushes, s.flushes) // copy because caller may modify after unlock
```

Defer unlocks at the end of methods when there's no early-return path — it reads more clearly than manual unlock.

### Performance Claims Require Production Evidence

Microbenchmarks alone are not sufficient justification for accepting or rejecting a change. Profile against a real or realistic cluster before making performance-based decisions.

> "I just checked profiles from one of our larger internal clusters and SortTrace is not present in CPU profiles at all... 0.02% of CPU."

A 3x slower implementation is acceptable if it's not on the hot path. Context matters more than benchmark numbers.

### Async Behavior Must Be Documented

Tempo enforces limits asynchronously in several components (live-store, block-builders). When a limit or behavior is enforced asynchronously rather than immediately, say so explicitly in both code comments and documentation.

### Minimal Exports

Default to unexported. Only export what callers actually need. If something is only used in tests, keep it in the test file and unexported.

> "I don't think it needs to be exported."
> "If this is just for testing, we could move it to that location and unexport it."

### Avoid Over-Engineering In-Process Calls

For monolithic or in-process communication, prefer a direct function call over gRPC, channels, or worker pools.

> "If this functionality is only for monolithic, let's avoid gRPC and push directly via a func."
> "If the push fails we can log the error and continue. No need to keep channels, workers, etc."

Match infrastructure complexity to the actual problem scope.

### Configuration Conventions

- Base scaling guidance on real cluster data, not round numbers
- Document why defaults changed in CHANGELOG and at the point of change in code

### Naming Must Be Precise

Names should convey unambiguous intent. When in doubt, ask: could this name mean something different to a reader unfamiliar with the context?

> "'Canonical' — I would expect canonical to parse/sort/rectify whitespace." (resolved to `NormalizeQuery`)

When editing an existing file, respect the naming style already present in that file over personal preference.
