#!/usr/bin/env bash
#
# Emit a JSON report of direct Go dependencies with the data needed
# to decide whether each one is stale: repo URL, pkg.go.dev version
# timeline (with deprecated/retracted flags), GitHub archived status,
# and blast radius.
#
# Usage:  check-dependencies.sh [project-dir]
# Output: JSON document on stdout, progress logs on stderr.

set -euo pipefail

# Number of deps to process concurrently. Each worker issues a few API calls.
# Default is conservative (4) to stay under pkg.go.dev's rate limits when the
# script is run several times in quick succession.
PARALLELISM=${PARALLELISM:-4}
# Module path prefixes that are skipped (trusted maintainers / large orgs).
SKIP_PREFIXES=${SKIP_PREFIXES:-"cloud.google.com
golang.org/x/
google.golang.org/
go.opentelemetry.io/
github.com/prometheus/
github.com/grafana/
github.com/Azure/
github.com/aws/
github.com/google/
github.com/hashicorp/
go.uber.org/
k8s.io/"}

PKGSITE=https://pkg.go.dev/v1beta

log() { printf '[check-deps] %s\n' "$*" >&2; }
die() { log "$@"; exit 1; }

# Check that each named command exists in PATH; die if any is missing.
require_cmd() {
    for c in "$@"; do
        command -v "$c" >/dev/null 2>&1 || die "missing required command: $c"
    done
}

# Return 0 if path starts with one of the trusted-org prefixes in $SKIP_PREFIXES.
is_skipped() {
    local path=$1 prefix
    while IFS= read -r prefix; do
        [ -z "$prefix" ] && continue
        case "$path" in
            "$prefix"*) return 0 ;;
        esac
    done <<<"$SKIP_PREFIXES"
    return 1
}

# Pseudo-versions are <base>-<14digit-timestamp>-<12hex-hash>, with two common shapes:
# - v0.0.0-YYYYMMDDHHMMSS-hash (no prior tag) and
# - v1.2.3-0.YYYYMMDDHHMMSS-hash (pre-release form, after a tag).
# Returns 0 (true) if the embedded timestamp is older than two years.
is_old_pseudo_version() {
    local version=$1 cutoff
    if [[ "$version" =~ ([0-9]{14})-[0-9a-f]{12}$ ]]; then
        cutoff=$(date -u -d '2 years ago' +%Y%m%d%H%M%S 2>/dev/null || date -u -v-2y +%Y%m%d%H%M%S)
        [ "${BASH_REMATCH[1]}" -lt "$cutoff" ]
    else
        return 1
    fi
}

# Emit a JSON array of suspicious-version flags (incompatible, old-pseudo-version).
scrutiny_flags_json() {
    local version=$1
    local flags=()
    if [[ "$version" == *+incompatible ]]; then
        flags+=("incompatible")
    fi
    if is_old_pseudo_version "$version"; then
        flags+=("old-pseudo-version")
    fi
    jq -nc --args '$ARGS.positional' "${flags[@]}"
}

# Construct the module path for the next semver major. Handles both Go-module
# style (foo/v8 -> foo/v9) and gopkg.in style (yaml.v3 -> yaml.v4). Modules
# without a major suffix return the /v2 candidate.
next_major_path() {
    local path=$1
    if [[ "$path" =~ ^(.+)/v([0-9]+)$ ]]; then
        printf '%s/v%d' "${BASH_REMATCH[1]}" "$((BASH_REMATCH[2] + 1))"
    elif [[ "$path" =~ ^(.+)\.v([0-9]+)$ ]]; then
        printf '%s.v%d' "${BASH_REMATCH[1]}" "$((BASH_REMATCH[2] + 1))"
    else
        printf '%s/v2' "$path"
    fi
}

CURL_TIMEOUT=${CURL_TIMEOUT:-30}

# Fetch <url> with retries differentiated by HTTP status:
# - 2xx → body to stdout, return 0
# - 429 (rate-limit) → sleep 10s and retry, up to 3 attempts total
# - 5xx / network error → sleep 2s and retry, up to 3 attempts total
# - 4xx (other than 429) → return 1 immediately (not retryable)
# Persistent failures return 1; the caller logs context.
curl_retry() {
    local url=$1 body_file code rc attempts=0
    body_file=$(mktemp "$WORK/curl-body.XXXXXX")
    while [ "$attempts" -lt 3 ]; do
        code=$(curl -sL --max-time "$CURL_TIMEOUT" -w '%{http_code}' \
            -o "$body_file" "$url" 2>/dev/null) || code='000'
        case "$code" in
            2*)   cat "$body_file"; rm -f "$body_file"; return 0 ;;
            429)  sleep 10 ;;
            4*)   rm -f "$body_file"; return 1 ;;
            *)    sleep 2 ;;
        esac
        attempts=$((attempts + 1))
    done
    rm -f "$body_file"
    return 1
}

# Resolve the upstream repo URL. github.com/<o>/<n>/... paths derive it
# directly; anything else (gopkg.in, vanity domains) goes through pkg.go.dev.
# Echoes the URL on success, empty on failure (failures logged to stderr).
resolve_repo_url() {
    local path=$1 raw
    if [[ "$path" =~ ^github\.com/([^/]+)/([^/]+) ]]; then
        printf 'https://github.com/%s/%s' "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}"
        return
    fi
    if raw=$(curl_retry "$PKGSITE/module/$path"); then
        printf '%s' "$raw" | jq -r '.repoUrl // empty' 2>/dev/null || true
    else
        log "pkgsite module $path: fetch failed (after retry)"
    fi
}

# Fetch the most recent versions from pkg.go.dev. Echoes the items array (with
# {version, commitTime, deprecated, retracted} per entry) on success, empty
# array if the module has no versions or the fetch failed (logged to stderr).
pkgsite_versions() {
    local path=$1 raw
    if ! raw=$(curl_retry "$PKGSITE/versions/$path?limit=20"); then
        log "pkgsite versions $path: fetch failed (after retry)"
        printf '[]'
        return
    fi
    printf '%s' "$raw" | jq -c '[.items // [] | .[] | {
        version: .version,
        commit_time: .commitTime,
        deprecated: (.deprecated // false),
        retracted: (.retracted // false)
    }]' 2>/dev/null || printf '[]'
}

# Returns "true" if a module at the next-major path is published on pkg.go.dev.
# 404 means the next major doesn't exist (the common case) — not logged.
# Honors 429 with a 10s backoff; other transient failures get a 2s backoff.
higher_major_exists() {
    local path=$1 next_path code attempts=0
    next_path=$(next_major_path "$path")
    while [ "$attempts" -lt 3 ]; do
        code=$(curl -sL --max-time "$CURL_TIMEOUT" -o /dev/null \
            -w '%{http_code}' "$PKGSITE/versions/$next_path?limit=1" 2>/dev/null) \
            || code='000'
        case "$code" in
            200|404|410) break ;;
            429)         sleep 10 ;;
            *)           sleep 2 ;;
        esac
        attempts=$((attempts + 1))
    done
    case "$code" in
        200)     printf 'true' ;;
        404|410) printf 'false' ;;
        *)       log "pkgsite higher-major $next_path: persistent status $code"
                 printf 'false' ;;
    esac
}

# Query gh for the GitHub repo's archived flag. Echoes "true"/"false" on
# success, "null" if the repo URL isn't a GitHub path or the call fails.
github_archived() {
    local repo_url=$1 archived
    if [[ "$repo_url" =~ ^https?://github\.com/([^/]+)/([^/]+?)(\.git)?$ ]]; then
        archived=$(gh api "repos/${BASH_REMATCH[1]}/${BASH_REMATCH[2]}" \
            --jq '.archived // false' 2>/dev/null) || archived='null'
        printf '%s' "$archived"
    else
        printf 'null'
    fi
}

# Count distinct modules that require the given module@version from the graph.
count_dependents() {
    local module_at=$1 graph_file=$2
    awk -v mod="$module_at" 'index($2, mod) == 1 { print $1 }' "$graph_file" | sort -u | wc -l | tr -d ' '
}

# Count vendored directories that import the given path. Uses -F so that
# regex metacharacters in the module path (e.g. `.`) aren't interpreted,
# and matches both the exact import and any subpackage so that the count
# doesn't pick up unrelated paths that share a prefix.
count_vendor_packages() {
    local path=$1
    [ -d vendor ] || { printf '0'; return; }
    grep -rl --include='*.go' -F -e "\"$path\"" -e "\"$path/" vendor/ 2>/dev/null \
        | awk 'NF { sub("/[^/]+$", "", $0); print }' | sort -u | wc -l | tr -d ' '
}

# Build the JSON record for a single direct dep and write it to $outfile.
# The three pkg.go.dev calls run sequentially within each dep (each ~2s) to
# cap total concurrent connections at $PARALLELISM rather than 3*$PARALLELISM,
# which avoids overloading pkg.go.dev and causing intermittent timeouts.
process_dep() {
    local idx=$1 path=$2 version=$3 outfile=$4
    local scrutiny repo_url versions archived higher_major dependents vendor_pkgs

    scrutiny=$(scrutiny_flags_json "$version")

    if is_skipped "$path"; then
        jq -n --arg path "$path" --arg version "$version" --argjson scrutiny "$scrutiny" \
            '{path: $path, version: $version, skipped: true,
              skip_reason: "trusted-prefix", extra_scrutiny: $scrutiny}' >"$outfile"
        return
    fi

    repo_url=$(resolve_repo_url "$path")
    versions=$(pkgsite_versions "$path")
    higher_major=$(higher_major_exists "$path")
    archived='null'
    if [ -n "$repo_url" ]; then
        archived=$(github_archived "$repo_url")
    fi
    dependents=$(count_dependents "$path@$version" "$WORK/graph.txt")
    vendor_pkgs=$(count_vendor_packages "$path")

    jq -n --arg path "$path" --arg version "$version" --argjson scrutiny "$scrutiny" \
        --arg repo_url "$repo_url" --argjson versions "$versions" \
        --argjson higher_major "$higher_major" --argjson archived "$archived" \
        --argjson dependents "$dependents" --argjson vendor_pkgs "$vendor_pkgs" \
        --arg repo_host "$([[ "$repo_url" =~ ^https?://github\.com/ ]] && echo github || echo other)" \
        '{
            path: $path,
            version: $version,
            skipped: false,
            extra_scrutiny: $scrutiny,
            repo_url: (if $repo_url == "" then null else $repo_url end),
            pkgsite: (if ($versions | length) == 0 then null else {
                latest_version: $versions[0].version,
                latest_commit_time: $versions[0].commit_time,
                deprecated: $versions[0].deprecated,
                retracted: $versions[0].retracted,
                higher_major_exists: $higher_major,
                recent_versions: $versions
            } end),
            github: (if $repo_host == "github" then {archived: $archived} else null end),
            blast_radius: {modules: $dependents, vendor_packages: $vendor_pkgs}
        }' >"$outfile"
}

main() {
    require_cmd go jq gh curl awk grep sort wc tr mktemp date

    local project_dir=${1:-.}
    cd "$project_dir"
    [ -f go.mod ] || die "no go.mod in $(pwd)"

    WORK=$(mktemp -d)
    trap 'rm -rf "$WORK"' EXIT

    log "project: $(go list -m 2>/dev/null || echo unknown)"

    go mod edit -json | jq -r '.Require[]? | select(.Indirect != true) | [.Path, .Version] | @tsv' \
        >"$WORK/deps.tsv"
    local total
    total=$(wc -l <"$WORK/deps.tsv" | tr -d ' ')
    log "found $total direct deps"

    log "computing go mod graph"
    go mod graph >"$WORK/graph.txt"

    mkdir -p "$WORK/results"
    log "processing deps with parallelism=$PARALLELISM"

    local idx=0
    while IFS=$'\t' read -r path version; do
        process_dep "$idx" "$path" "$version" "$WORK/results/$idx.json" &
        idx=$((idx + 1))
        if [ "$((idx % PARALLELISM))" -eq 0 ]; then
            wait
        fi
    done <"$WORK/deps.tsv"
    wait

    log "merging results"
    local module
    module=$(go list -m 2>/dev/null || echo "")
    jq -s --arg module "$module" --arg generated_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        '{generated_at: $generated_at, module: $module, deps: .}' "$WORK/results"/*.json
}

main "$@"
