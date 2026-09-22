---
name: ci-health-audit
description: Audit grafana/tempo CI for flaky jobs and slow jobs using gh and action logs and data, then fix what the evidence supports (flaky test, matrix split) and flag what it doesn't (runner capacity, alreay fixed issues, etc).
allowed-tools: Bash Read Edit Grep Glob
---

# CI Health Audit

Find flaky and slow CI jobs in `grafana/tempo` from real run data, fix what's confirmed, flag what isn't.

## Before starting

- Local checkouts often point `origin` at a personal fork, not `grafana/tempo` upstream. Confirm which repo you're targeting before running any `gh` command.
- Every `gh` command needs `--repo grafana/tempo` (or use `gh api repos/grafana/tempo/...` directly), or it queries the wrong repo silently.
- Pick a window: 4 weeks of runs is usually enough data and keeps API calls under GitHub's rate limit.
- `ci.yml` runs 100+ times/week - don't pull every run. Sample recent runs, and weight sampling toward failed/rerun runs since that's where flaky-job evidence lives.
- Check `git branch --show-current` before you start editing, and again right before staging or committing. A local checkout can be shared with concurrent work happening in the same directory. If the branch isn't what you expect, stop and ask - don't guess what happened.
- Steps 1 and 2 are independent data-gathering passes and can be split across parallel sub-agents when the window is large, with you consolidating and re-verifying their claims (see step 2) rather than trusting a summary as-is. Data gathering parallelises. **Measurement does not** - see step 3.
- This skill only covers CI config inside this repo - a fix that would require changing anything outside it is out of scope. Flag it instead.
- `-race` is required in CI and is not negotiable. Never propose removing, disabling or conditionally skipping it as a speedup. You may run without it locally as a diagnostic (step 3), but the fix always has to work with it on.

## Workflow

### 1. Find slow jobs

First ask the user for a CSV export from GitHub's own Actions Performance dashboard: `https://github.com/grafana/tempo/actions/metrics/performance?tab=jobs`, Jobs tab, exported for the window you want. It gives per-job avg run time, avg queue time, and failure rate across every run in the window in one file - far less work than sampling, and it surfaces failure rate directly (useful context for step 2 too). This page needs an authenticated browser session behind it - both WebFetch and `gh api` return a 404 for it, so it cannot be fetched automatically. If the user can provide it, use it as the primary data source for this step and skip the sampling below.

If no CSV is available, fall back to sampling live runs:

```bash
gh api "repos/grafana/tempo/actions/workflows/ci.yml/runs?per_page=100&status=completed&created=>=<date>" --paginate
```

Sample 40-60 `pull_request`-triggered completed runs - recent ones are fine, you don't need every run in the window. For each, pull job-level timing:

```bash
gh api "repos/grafana/tempo/actions/runs/<run_id>/jobs?per_page=100"
```

Compute per-job-name mean/median/max duration from `started_at`/`completed_at`. Then compute **start delay**: each job's `started_at` minus the run's earliest job `started_at`. A job with normal duration but occasional huge start-delay outliers means runner-queue contention, not slow execution - don't conflate the two. Check whether `needs:` chains actually cost real time (they usually don't if the upstream job is cheap).

Then do two things before optimising anything, in this order.

**Name the critical path.** Run time is the duration of the slowest job. Rank the jobs, identify the one that sets it, and when you propose a change say whether it reduces run time or only compute cost. Work on a job that was never on the critical path does not move the headline number.

**Split each slow job into setup versus test execution.** Compare the step's wall duration against the test runner's own reported total, which `gotestsum` prints as `PASS Package x/y (4m46s)`. The remainder is setup: tooling install, module download, compilation. It is routinely minutes and invisible if you only look at test durations, so check it before profiling anything. Two things worth checking: whether the `make` target pulls in more tooling than the tests actually need, and whether the Go build cache is being re-saved rather than restored from a key that never changes.

### 2. Find flaky jobs

Pull failed/cancelled runs, plus successful runs with `run_attempt > 1` (someone had to rerun - that's a flake signal even though the final state is green). For each distinct failing job, get the real log - don't guess from the job name:

```bash
gh api "repos/grafana/tempo/actions/jobs/<job_id>/logs" --allow-escape-sequences
```

**The strongest evidence is same-commit, same-test, multiple attempts.** If a job failed on attempt 1 and attempt 2 of the same SHA with the identical error, that's a confirmed flake, not a one-off:

```bash
gh api "repos/grafana/tempo/actions/runs/<run_id>/attempts/<n>/jobs?per_page=100"
```

Do not trust a grouped or summarized claim without re-reading the actual log - ground every claim in the log from an actual run, not in a label like "same job family."

Before proposing a fix, check whether the failure is already fixed on a recent merged PR:

```bash
git log --oneline --all -S '<distinctive error substring or image name>' -- '<relevant path>'
gh pr view <n> --repo grafana/tempo --json mergedAt,body
git merge-base --is-ancestor <fix-commit> <pr-head-sha>   # confirms whether a given PR's branch already has the fix
```

If it already landed before the window you're auditing, say so and move on - don't re-fix it.

### 3. Measure a slow test honestly

Most wrong numbers come from this step.

- Benchmark one thing at a time, and record the machine load you measured under.
- Prefer counts to times. Counts are deterministic and survive a loaded machine.
- Verify a pprof percentage before believing it. A sample share is not CPU, and a cumulative total is not a function's own cost - cross-check against measured `sys` time and `pprof -peek`.
- For times: interleave before and after, several reps, `GOMAXPROCS` pinned to the runner's core count, report CPU rather than wall.
- Express a saving as a fraction of what a quiet runner spends, not of a loaded local total.
- Don't claim an effect smaller than your run-to-run spread. Say you can't measure it.
- Tests run under `-race`, which charges per synchronisation event, so on concurrent code target the number of goroutines and channel operations rather than bytes or CPU efficiency.
- Confirm the binary or image under test was built from your branch before trusting any number from it.

### 4. Fix a confirmed flaky test

Once a flake is confirmed (same commit, same test, multiple attempts, real log evidence from step 2), fix it - don't stop at reporting it.

1. Read the failing test and trace the assertion back to what it depends on. Find why it's non-deterministic (a race against eventual consistency, a fixed sleep, shared test state) - don't guess.
2. Look for an existing pattern elsewhere in the same file or package that already handles the same kind of non-determinism correctly, and reuse it instead of inventing new synchronization. Keeps the fix minimal and consistent with the codebase.
3. Build and test-compile after editing (`go build ./...`, `go vet ./...`, `go test -run ^$ ./...` on the changed package) to catch compile errors the fix introduces.
4. Verify by rerunning the specific test multiple times, not once, with a timeout generous enough for all iterations combined. Confirm the actual pass/fail result rather than trusting a truncated tail of the output.
5. If the flake is actually an external dependency (image registry, network) rather than test logic, the fix belongs in the harness/test setup, not the assertion - check step 2's "already fixed" search first, since this class of fix is often already merged.

### 5. Rules for any speedup you propose

- Say what it stops verifying, in counts, not adjectives. If you can't name what's lost, you don't understand the change.
- Never speed up the test harness at production's expense. Measure any production-code change with and without `-race` and report both, since a knob can look good under the detector and be slower in production.
- Don't pattern-match a config value across tests. The same constant can be pathological in one fixture and irrelevant in another. Screen by absolute impact, size times how often it's paid, and measure the specific case.

### 6. Find precedent for splitting

If a job is genuinely too slow and can't be sped up in place:

```bash
git log --oneline --all --full-history -- '.github/workflows/ci.yml'
git show <sha> -- .github/workflows/ Makefile
gh pr view <n> --repo grafana/tempo --json body
```

This repo's established pattern (seen across multiple past splits, e.g. `tempodb-wal` carved out of `tempodb`, the integration-tests folder-per-matrix-entry rewrite): one Makefile target per logical test group, one matrix entry per Makefile target, `fail-fast: false`. Follow it - don't invent a new mechanism (sharding flags, `-p` parallelism) unless the user asks for one.

If splitting integration tests specifically, check `integration-tests-validation` in `ci.yml` - it enforces folder<->matrix-entry parity and fails loudly if you add a folder without wiring it in.

### 7. Implement a matrix split

1. Identify the biggest sub-bucket inside the oversized target - by file count first, then confirm with a real timing run, not a guess.
2. Add a path exclusion to the old target's source variable, and add a new target that includes exactly that path, using the same pattern as the existing targets.
3. Add the new target name to the matrix array in `ci.yml`.
4. Verify the partition is exact: the new target's packages plus the old target's remaining packages should equal the original set, with no overlap and no gap. If a verification command comes back suspiciously empty for a target you didn't even touch, suspect environment noise (a stray local git worktree, an extra module, a broken tool cache) before concluding your edit is wrong - rerun the same check as a control against an untouched target first.
5. Run the new target's actual test command (not a dry-run) and confirm it passes with real output, not just a clean exit code.

### 8. Flag runner-queue starvation

If duration data shows jobs occasionally queuing for tens of minutes to hours while sibling jobs already finished, that's runner capacity or org-wide contention, not something a workflow file in this repo can fix. Confirm whether you have the access to diagnose further (e.g. an org-admin-only API) - if not, flag it for the user to escalate instead of guessing at a workflow change. Don't ship an unverified fix for any problem - ask for more detail when you need it.

### 9. Reorganize test packages (rarely worth it)

Do this only when the user asks for it, or when a package boundary genuinely blocks a split. It is real work that typically buys no run-time improvement on its own, so never present it as a speedup. If you do it:

Map cross-file dependencies before drawing package boundaries - don't guess from file names alone.

Keep a structurally slow test grouped with its topic. Fix the speed inside the test itself rather than changing shared test infrastructure - lower blast radius, even for an opt-in change.

Check shared test setup for assumptions tied to package location before restructuring folders.

Splitting one file into several for readability doesn't require a CI or build change.

Diff any moved or split file against its last committed version before trusting it.

## Final Response

Report:
- Every flaky job/test claim: job name, error text quoted from a real log, run/job IDs as evidence, whether same-commit reruns confirm it.
- Every slow-job claim: sample size, mean/median/max in real units.
- Every "already fixed" or "can't fix here" item, stated explicitly rather than padded with a guess.
- Any splitting precedent cited: the actual PR number and what it changed, not just a description of the pattern.
- Any fix made: file changed, what verification confirmed it works.
- For every speedup: whether it reduces run time or only compute cost, and what it stopped verifying.
- Every number's basis: measured or estimated, under what machine load, and with `-race` on or off. Retract rather than defend a figure you can no longer support.
