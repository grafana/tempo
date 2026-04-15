---
name: fix-vendor-conflicts
description: Resolve vendor/ conflicts during a merge, rebase, or dependency upgrade on main or release branches
allowed-tools: Bash Read Grep Edit
---

# Fixing Vendor Conflicts

Use this skill when resolving conflicts in `vendor/` during a merge, rebase, or dependency upgrade.

## Before starting

**Ask the user:** are you working on the **main** branch or a **release** branch? The answer changes what kinds of fixes are acceptable (see below).

## What this skill does

Guides the process of diagnosing and resolving vendor directory conflicts caused by dependency changes — whether a direct bump or a transitive cascade.

---

## Core principles

**Never patch vendor/ files manually.** CI runs `go mod vendor` and will overwrite any manual patches. All fixes must come from upgrading or pinning the upstream dependency to a version that has the fix already.

**Always run the pre-commit checklist before committing anything:**

```bash
go mod tidy
go mod vendor
go build -mod vendor ./...
go test -mod vendor -run ^$ ./...
go run pkg/docsgen/generate_manifest.go
```

All must pass with no errors. If `generate_manifest.go` produces changes, commit them:

```bash
git add docs/sources/tempo/configuration/manifest.md && git commit -m "manifest.md"
```

`go build` catches compile errors in production code; `go test -run ^$` catches compile errors in test and integration files. Run these after every change — dependency upgrades, cherry-picks, and manual fixes alike — before staging and committing.

---

## Step-by-step process

### 1. Understand what changed

```bash
git diff go.mod
```

Note which module(s) were bumped. A single bump (e.g. grpc) often cascades into failures in transitive dependencies.

### 2. Check what main did

Main branch has almost always already solved the problem. Find relevant commits:

```bash
git log --oneline main | grep -i "<module-name>"

# Check what dependency versions main used at a given commit
git show <commit-sha>:go.mod | grep -E "<module-name>|<related-module>"

# Inspect a specific vendor file at a commit
git show <commit-sha>:vendor/<path/to/file.go>
```

Use main as the source of truth for what versions to target and what code changes are needed.

### 3. Run tidy to surface errors

```bash
go mod tidy
```

This will either succeed or tell you exactly what is missing or mismatched. Read the error trace carefully — it shows the import chain that led to the conflict.

### 4. Fix each error

For each error `go mod tidy` surfaces:

- If it's a **missing package in a module**: a transitive dep requires a version of that module that doesn't have the package. Upgrade (or pin) the module to a version where the package exists or is no longer needed.
- If it's a **compile error from an interface change**: a dependency updated an interface. Find the implementing type in this repo and update it to match. Check main for the exact fix.
- If it's a **call-site mismatch** (wrong number of args, wrong signature): the upstream library changed a function signature. Update all callers, mocks, and test helpers.

Always check main at the relevant commit before writing new code:

```bash
git show <commit-sha> -- <path/to/file.go>
```

### 5. Re-run tidy, then vendor

```bash
go mod tidy
go mod vendor
```

Both must exit 0. If `go mod vendor` fails with a missing package, try `go mod download` first, then re-run vendor.

### 6. Verify fixes came from upstream

After vendoring, confirm that your fixes are present in vendor/ as real upstream code — not patches:

```bash
grep -A5 "<relevant symbol>" vendor/<path/to/file.go>
```

If you only see a stub or `Unimplemented*` type, the module version is too old. Upgrade to a newer one.

### 7. Build to confirm no remaining compile errors

```bash
go build -mod vendor ./...
```

This compiles every package in the repo, not just the main binary entrypoint. `make tempo` only builds `cmd/tempo` and will miss compile errors in other packages (e.g. `modules/distributor`). Always use `./...` to catch issues across the full project.

### 8. Check test files compile

`go build` does not compile `_test.go` files. Run this to catch errors in test code too:

```bash
go test -mod vendor -run ^$ ./...
```

The `-run ^$` flag matches no test names, so no tests execute — it only compiles. Test files often contain mock types that implement interfaces, and a dependency upgrade that adds a method to an interface will break those mocks without showing up in `go build`.

If new failures appear, find the commit on main that introduced the fix and evaluate whether to cherry-pick or fix manually (see below).

Also run this specifically for integration test directories, which are often missed:

```bash
go test -mod vendor -run ^$ ./integration/...
```

Ask the user if they want to run broader test coverage (e.g. `make test`, `make test-e2e`). These take a long time and are not required for every commit, so only run them if the user confirms.

**If new errors appear after any step, loop back to step 3** (`go mod tidy`) and work through the errors again. Cascading dependency conflicts often require multiple passes.

### Cherry-pick vs manual fix

**Prefer cherry-pick when:**
- The commit on main is small and targeted (e.g. "fix interface X", "add missing method")
- The commit touches only the files you need changed
- Cherry-picking it won't pull in unrelated new files, new dependencies, or feature code

**Prefer a manual fix when:**
- The only commit on main that has the fix is a large feature commit — it adds new production code, new integration tests, or upgrades additional dependencies as side effects
- Cherry-picking would require follow-up work (reverting unwanted files, upgrading more deps) that outweighs the benefit
- The fix itself is trivial (e.g. a one-line method stub on a mock type)

When making a manual fix on a release branch, keep it to the absolute minimum needed to satisfy the compiler. This only applies to **test mock types in `_test.go` files** — never to production code. Typically a stub that panics, matching the pattern of other unimplemented methods in the same mock type:

```go
func (r mockRing) GetSubringForOperationStates(_ ring.Operation) ring.ReadRing {
    panic("implement me if required for testing")
}
```

If you cherry-picked a commit and later discover it brought in too much, revert cleanly and apply the manual fix instead:

```bash
git revert --no-commit <cherry-pick-sha> [<follow-up-sha> ...]
git commit -m "revert: undo cherry-pick of <sha> (<reason>)"
# then apply the minimal fix manually
```

### 9. Regenerate the config manifest

After all changes are in, regenerate the configuration manifest so the docs stay in sync with the actual config:

```bash
go run pkg/docsgen/generate_manifest.go
```

This updates `docs/sources/tempo/configuration/manifest.md`. Commit the result if anything changed:

```bash
git add docs/sources/tempo/configuration/manifest.md && git commit -m "manifest.md"
```

---

## Common patterns of cascading conflicts

When one module is bumped, look for these patterns:

- **Interface additions**: a core library adds a method to an interface your code implements → add the method to your type
- **Signature changes**: a library changes the signature of a function → update all callers, including mocks and test helpers
- **Replace directive staleness**: a `replace` directive in `go.mod` pins a fork to an old version that's incompatible with the new dep → update the replace directive
- **Test-only import conflicts**: `go mod tidy` resolves test dependencies too; a module's tests may import an internal package that was removed in a newer version → upgrade that module to a version where its tests no longer import the removed package

---

## Main vs release branch

### Main branch
Manual code changes are acceptable — if a dependency upgrade requires updating an interface implementation, a function signature, or call sites in this repo, go ahead and make those changes directly.

### Release branch
**Do not make manual production code changes.** A release branch should only receive targeted backports, not new hand-written source changes driven by dependency upgrades. Instead, find the commit on main that already contains the fix and cherry-pick the relevant source file changes from it.

The one exception is a **minimal test-only compatibility stub** in a `_test.go` file when no suitable fix exists on main and the change is required solely to satisfy the compiler for test code. Keep it to the absolute minimum — avoid changing runtime behavior.

#### Finding the right commit to cherry-pick

For each module that requires a code fix:

1. Identify which module is causing the issue and what version change triggered it (e.g. `github.com/grafana/gomemcache` v0.0.0-20250228 → v0.0.0-20251008 added `ctx` to `GetMulti`).

2. Use `git log -L` to trace commits on main that touched that module's line in `go.mod`:
```bash
git log --oneline -L '/github.com\/grafana\/gomemcache/,+1:go.mod' main
```

3. Walk backwards through the commits. The most recent commit touching that line may not have the fix — go one or two commits earlier. Look for a commit whose description mentions an interface or signature change (not just a version bump).

4. Verify the commit has the fix you need:
```bash
git show <commit-sha> -- <path/to/file.go>
```

#### Applying the cherry-pick on a release branch

Commit the go.mod/vendor changes first, then do a regular `git cherry-pick` (no `--no-commit`). This preserves the original commit message and author, making it clear to PR reviewers that the source code changes were cherry-picked from main rather than written manually.

```bash
# 1. Commit all the go.mod, go.sum, and vendor/ changes first
git add -A && git commit -m "chore(deps): update <modules>"

# 2. Cherry-pick — this will pause on conflicts
git cherry-pick <commit-sha>
```

The cherry-pick will conflict on `go.mod`, `go.sum`, and `vendor/` — because we already have the newer versions from step 1. **Keep ours** for all of those. The source file changes (interface implementations, call sites) will usually auto-merge cleanly.

For any new doc or config files added by the dep upgrade (e.g. a generated config manifest), **take theirs** — these reflect the new capabilities of the upgraded dependency.

```bash
# Keep our go.mod/vendor (already up to date)
git checkout --ours go.mod go.sum vendor/modules.txt vendor/<affected-module>/

# Take theirs for any newly added doc/config files
git checkout --theirs docs/sources/tempo/configuration/manifest.md

# Stage the resolved files
git add go.mod go.sum vendor/modules.txt vendor/<affected-module>/ docs/...

# Verify no conflicts remain, then continue
git diff --name-only --diff-filter=U  # should be empty
git cherry-pick --continue --no-edit
```

The resulting commit will carry the original message and author from main, e.g.:
```
Update dskit and adjust memcached interface signature (#5604)
Author: Zach Leslie <zach.leslie@grafana.com>
```

---

## Notes for release branch backports

- Always check main first — it has already solved the problem, usually at a specific commit
- A single module bump can require changes to several related modules; fix them together
- The `go mod tidy` error trace is the best guide to what needs fixing — read it fully before acting
