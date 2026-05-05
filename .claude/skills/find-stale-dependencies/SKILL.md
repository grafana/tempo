---
name: find-stale-dependencies
description: Find stale direct Go dependencies by checking repository status, README, blast radius, and usage context, then output a prioritized list
allowed-tools: Bash, WebFetch, Grep, Read
---

# Find Stale Go Dependencies

Scans all **direct** dependencies in `go.mod` for signs of abandonment and ranks them by risk.

A dependency is **direct** if its line does NOT end with `// indirect`.

## Staleness criteria (any one qualifies)

- Repository is **archived**
- README says: *deprecated*, *unmaintained*, *no longer maintained*, *use X instead*
- Latest commit (or last release for the depended-on major) is **older than 2 years**
- Version itself is suspect (pinned to a >2-year-old pseudo-version, or `+incompatible`)

## Step 1 — Run the data-collection script

The skill ships with a script that gathers all deterministic facts about the project's direct dependencies in one pass: repo URL, GitHub archived/pushed_at, last release for the major, version-string smells, and blast radius.

```bash
.claude/skills/find-stale-dependencies/check-dependencies.sh > /tmp/deps-report.json
```

The script:
- Lists direct deps from `go mod edit -json`
- Marks deps under trusted-org prefixes (`golang.org/x/`, `github.com/grafana/`, `github.com/google/`, `k8s.io/`, …) as `skipped`
- Resolves vanity-domain modules (`go.yaml.in/`, `gopkg.in/`, …) via `go mod download -json` so non-GitHub origins are handled too
- Flags suspect version strings (`+incompatible`, old pseudo-versions) via `extra_scrutiny`
- Computes blast radius from `go mod graph` and `vendor/`

Override the trusted-org list with `SKIP_PREFIXES=$'pfx1\npfx2'` if needed. Adjust parallelism with `PARALLELISM=N`.

The output JSON has shape:

```json
{
  "generated_at": "2026-05-05T12:00:00Z",
  "module": "github.com/grafana/tempo",
  "deps": [
    {
      "path": "github.com/foo/bar",
      "version": "v1.2.3",
      "skipped": false,
      "extra_scrutiny": ["incompatible" | "old-pseudo-version"],
      "repo_url": "https://github.com/foo/bar",
      "github": {
        "owner": "foo",
        "name": "bar",
        "archived": false,
        "pushed_at": "2024-01-01T00:00:00Z",
        "recent_releases": [
          {"tag": "v1.2.3", "date": "2024-01-01T00:00:00Z"}
        ]
      },
      "blast_radius": {"modules": 12, "vendor_packages": 3}
    }
  ]
}
```

`github` is `null` for non-GitHub origins (e.g., gopkg.in vanity paths the proxy doesn't resolve); `repo_url` carries the raw URL when available. `recent_releases` falls back to tag names with `date: null` when the repo publishes tags but no GitHub Releases.

`skipped: true` deps still carry their path/version but no other data — surface them only if `extra_scrutiny` is non-empty.

## Step 2 — Pick the candidate set from the JSON

A dep is a **stale candidate** if any of these is true:

- `github.archived == true`
- `github.pushed_at` is older than **today − 2 years**
- `github.recent_releases` contains a release for a higher major than the path's `/vN` suffix (the depended-on major is frozen, a successor exists)
- `extra_scrutiny` is non-empty

Filter with jq:

```bash
TWO_YEARS_AGO=$(date -u -d '2 years ago' +%Y-%m-%dT%H:%M:%SZ)
jq --arg cutoff "$TWO_YEARS_AGO" '
  def path_major:
    ((capture("/v(?<n>[0-9]+)$") | .n)? // "1") | tonumber;
  def release_major(t):
    (t | capture("^v(?<n>[0-9]+)\\.").n | tonumber)? // -1;

  .deps | map(select(
    (.skipped | not) and (
      .github.archived == true
      or (.github.pushed_at != null and .github.pushed_at < $cutoff)
      or (
        (.path | path_major) as $m
        | (.github.recent_releases // []) | any(release_major(.tag) > $m)
      )
      or ((.extra_scrutiny // []) | length > 0)
    )
  ))
' /tmp/deps-report.json
```

Skipped deps are also worth a second look when `extra_scrutiny` flags fire:

```bash
jq '.deps | map(select(.skipped and ((.extra_scrutiny // []) | length > 0)))' /tmp/deps-report.json
```

Stop checking a dep as soon as one criterion fires — no need to keep verifying.

## Step 3 — Confirm via README (LLM judgment)

For each candidate, fetch the README and look for explicit deprecation signals. Use `gh` for GitHub repos, `WebFetch` otherwise:

```bash
gh api repos/<owner>/<repo>/readme --jq '.content' | base64 -d | head -60
```

Confirming phrases:

- *deprecated*, *unmaintained*, *no longer maintained*, *archived*
- *use X instead*, *successor is X*, *replaced by X*, *moved to X*
- *not actively maintained*, *paused maintenance*, *deprecation mode*

Record whether the README confirms staleness independently of the JSON signals — it strengthens the recommendation.

## Step 4 — Identify a replacement (LLM judgment)

For each confirmed-stale dep:

1. Extract the recommended replacement from the README (phrases like *use X instead*, *migrate to*, *successor is*) or from a `pkg.go.dev` "moved to" notice.
2. Check whether it's already in `go.mod`:

```bash
go mod edit -json | jq -r '.Require[].Path' | grep -Fx "<replacement-path>"
```

Report **replacement already in go.mod** (migration effort only) vs **replacement not yet added** (requires dep bump + migration). If no replacement is identifiable, say so.

## Step 5 — Determine usage context (LLM judgment)

Search the codebase for direct call sites:

```bash
grep -rn "\"<import-path>" --include='*.go' . | head
```

Classify the usage as one of:

- **Hot path**: request/response handling, payload marshal/unmarshal, HTTP or gRPC middleware, authentication, networking stack
- **Internal utility**: CLI tools, build helpers, internal logging, test fixtures
- **Test-only**: only imported under `_test.go` files

This is what the JSON cannot give you — it requires reading the code.

## Step 6 — Classify priority

Use this decision order:

1. **High** — confirmed deprecated/archived **and** on the hot path, **or** confirmed deprecated/archived **and** a maintained successor exists (migration is the obvious next step)
2. **Medium** — quietly abandoned (>5 years since last activity, no explicit deprecation notice) but only used in internal utilities
3. **Low** — stale but only used internally and last activity within the last 5 years

The skill does not assign a Critical tier — vulnerabilities are handled by other tooling (govulncheck/Dependabot). If you do see a CVE-driven emergency on a stale dep, that's a security-tooling alert, not a staleness report.

## Step 7 — Report

Skip non-stale dependencies. Output only stale ones, sorted by priority:

```
## Stale Dependencies

### High
- `import/path@version` — reason: <archived | README says deprecated | last commit YYYY-MM-DD | old-pseudo-version>
  Blast radius: N modules, M vendor packages
  Replacement: `suggested/path` [already in go.mod | not yet added | none]
  Recommendation: <replace with X | upgrade | remove>

### Medium
[same format]

### Low
[same format]
```

## Efficiency notes

- The script does all the deterministic data collection in one pass — do not re-run individual `gh api` / `go mod` commands per dep
- For LLM-only steps (README, usage context), batch the work: fetch all confirmed-stale READMEs in parallel `gh api` calls, then do all `grep` searches in parallel
- Stop investigating a dep once one staleness signal is confirmed
