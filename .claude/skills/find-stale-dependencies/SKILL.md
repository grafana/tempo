---
name: find-stale-dependencies
description: Find stale direct Go dependencies by checking repository status, README, CVEs, blast radius, and usage context, then output a prioritized list
allowed-tools: Bash, WebFetch, Grep, Read
---

# Find Stale Go Dependencies

Scans all **direct** dependencies in `go.mod` for signs of abandonment and ranks them by risk.

A dependency is **direct** if its line does NOT end with `// indirect`.

## Staleness criteria (any one qualifies)

- Repository is **archived**
- README says: *deprecated*, *unmaintained*, *no longer maintained*, *use X instead*
- Latest commit is **older than 2 years**

## Step 1 — Extract direct dependencies

Run this command to get all direct deps with versions (`.Indirect` is absent for direct deps, not `false`):

```bash
go mod edit -json | jq -r '.Require[] | select(.Indirect != true) | .Path + " " + .Version'
```

Fallback without `jq`:
```bash
awk '/^require\s*\(/{p=1} p && !/\/\/ indirect/ && /^\t[a-z]/{print $1, $2} /^\)/{p=0} /^require [a-z]/ && !/\/\/ indirect/{print $2, $3}' go.mod
```

**Skip these prefixes** (actively maintained by large orgs):
`cloud.google.com`, `golang.org/x/`, `google.golang.org/`, `go.opentelemetry.io/`, `github.com/prometheus/`, `github.com/grafana/`, `github.com/Azure/`, `github.com/aws/`, `github.com/google/`, `github.com/hashicorp/`, `go.uber.org/`, `k8s.io/`

**Extra scrutiny candidates** — check these even if the org is on the skip list:
- Pseudo-versions (`v0.0.0-YYYYMMDD-hash`): if the embedded date is >2 years old, likely stale
- Packages with `+incompatible` suffix: pre-modules era, often from inactive repos

## Step 2 — Resolve the repository URL

For `github.com/<owner>/<repo>/...` imports, the repo is `https://github.com/<owner>/<repo>`.

For **all other imports** (vanity domains like `go.yaml.in/`, `gopkg.in/`, custom domains):
1. Resolve the source repository URL via the module proxy (module is usually already cached in vendor repos):
   ```bash
   go mod download -json <import-path>@<version> | jq -r '.Origin.URL // empty'
   ```
2. If `.Origin.URL` is empty, fall back to `WebFetch` of `https://pkg.go.dev/<import-path>` and look for the source link.
3. If both methods fail to produce a URL, **skip the dep and note it explicitly in the report** as "URL could not be resolved — skipped". Do not silently drop it.
4. Use the resolved GitHub URL for all subsequent checks.

Do not assume a vanity domain hosts its own repo — always resolve it first.

## Step 3 — Check repository staleness

Use the `gh` CLI for GitHub repos (issue multiple calls in parallel):

```bash
# Check archived status and last push date
gh api repos/<owner>/<repo> --jq '{archived: .archived, pushed_at: .pushed_at}'

# Fetch the README (first 60 lines is enough)
gh api repos/<owner>/<repo>/readme --jq '.content' | base64 -d | head -60
```

If `gh` is unavailable, fall back to `WebFetch` of `https://github.com/<owner>/<repo>`.

**Mark as stale if any of these are true:**
- `archived: true`
- README contains: *deprecated*, *unmaintained*, *no longer maintained*, *archived*, *use X instead*, *not actively maintained*, *paused maintenance*, *deprecation mode*
- `pushed_at` is before **[today minus 2 years]**

**Stop checking a dep once any one staleness signal fires** — no need to verify all three.

**Major-version caveat**: `pushed_at` reflects activity across all branches and versions. For repos with multiple major versions, a recent `pushed_at` does not mean the version you depend on is maintained. Apply these additional checks:

- **Versioned import paths** (`/v2`, `/v8`, etc.): extract the major N from the path and check the last release for that specific major (use `per_page=100` — `--paginate` doesn't work with `--jq`):
  ```bash
  MAJOR=$(echo "<import-path>" | sed -n 's|.*/v\([0-9][0-9]*\)$|\1|p'); MAJOR=${MAJOR:-1}
  gh api "repos/<owner>/<repo>/releases?per_page=100" --jq \
    "[.[] | select(.tag_name | startswith(\"v${MAJOR}.\"))] | sort_by(.published_at) | last | .published_at"
  ```
  If the last vN release is >2 years old while a higher major exists → stale.

- **Unversioned imports** (v0/v1): check whether a `/v2` successor exists in the same repo:
  ```bash
  gh api "repos/<owner>/<repo>/releases?per_page=100" --jq \
    '[.[] | select(.tag_name | test("^v[2-9]"))] | length'
  ```
  If a higher major exists and v1 README or tags show no recent activity → likely frozen; check README for deprecation language.

Also record whether the dep is **quietly abandoned**: last release or `pushed_at` before **[today minus 5 years]** with no explicit deprecation notice. Used for priority classification in step 7.

## Step 4 — Check for CVEs

For each confirmed stale dep, fetch in parallel:
```
https://pkg.go.dev/search?m=vuln&q=<url-encoded-import-path>
```
If vulnerability entries appear → mark as CVE-affected.

## Step 5 — Find replacement

For each stale dep, identify the recommended replacement:
1. **README**: extract the suggested import path from phrases like "use X instead", "migrate to", "replaced by", "successor is"
2. **pkg.go.dev**: check if the module page shows a "moved to" or "redirects to" notice

If a replacement is found, check whether it's already in `go.mod`:
```bash
grep "^	<replacement-path> " go.mod
```
Report: **replacement already in go.mod** (migration effort only) vs **replacement not yet added** (requires dep bump + migration).

## Step 6 — Measure blast radius

For each stale dep, run these in parallel:

```bash
# Modules that depend on it (from the full dependency graph)
go mod graph | awk -v mod="<module>@" '$2 ~ "^" mod { print $1 }' | sort -u | wc -l

# Vendor packages that import it directly
grep -r '"<import-path>' vendor/ --include="*.go" -l 2>/dev/null | xargs -r dirname | sort -u | wc -l
```

Report both counts in the output.

## Step 7 — Classify priority

Assign priority using this decision order:

1. **Critical** — has CVEs (from step 4)
2. **High** — used in request/response handling, payload marshal/unmarshal, HTTP or gRPC middleware, authentication, or the networking stack
3. **Medium** — quietly abandoned (>5 years since last commit, no explicit deprecation notice) but used only in internal utilities, CLI tools, or tests
4. **Low** — stale but only used internally and last commit <5 years ago

## Step 8 — Report

Skip non-stale dependencies. Output only stale ones, sorted by priority:

```
## Stale Dependencies

### Critical
- `import/path@version` — reason: <archived | README says deprecated | last commit YYYY-MM-DD>
  CVEs: <IDs or link>
  Blast radius: N modules, M vendor packages
  Replacement: `suggested/path` [already in go.mod | not yet added]
  Recommendation: <replace with X | upgrade | remove>

### High
[same format, omit CVEs line]

### Medium
[same format]

### Low
[same format]
```

## Efficiency notes

- **Parallel batching**: issue all `gh api` staleness checks for multiple repos in one turn; do the same for CVE fetches and blast radius greps
- **Fail fast per dep**: once a staleness signal fires, skip remaining checks for that dep and move to replacement + blast radius
- **CVE and blast radius in one batch**: after confirming staleness, fetch CVEs and measure blast radius simultaneously for all confirmed-stale deps
