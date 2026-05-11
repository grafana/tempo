#!/usr/bin/env bash
#
# Emit a JSON report of direct Go dependencies with the data needed
# to decide whether each one is stale: repo URL, GitHub status,
# recent releases (or tags), and blast radius.
#
# Usage:  check-dependencies.sh [project-dir]
# Output: JSON document on stdout, progress logs on stderr.

set -euo pipefail

# Number of deps to process concurrently. Each worker issues a few gh api calls.
PARALLELISM=${PARALLELISM:-8}
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

# Echo the upstream repo URL for a module via go mod download, or empty.
resolve_repo_url() {
    local path=$1 version=$2
    go mod download -json "$path@$version" 2>/dev/null | jq -r '.Origin.URL // empty' 2>/dev/null || true
}

# Fetch GitHub metadata + the most recent 20 releases (falling back to tag
# names when the repo doesn't publish releases — date is null in that case).
# Records which API calls failed in `github_errors` so the consumer can
# distinguish "repo healthy" from "data not available".
github_meta() {
    local owner=$1 name=$2 meta releases errs_json
    local errs=()

    if ! meta=$(gh api "repos/$owner/$name" --jq '{archived, pushed_at}' 2>/dev/null); then
        meta='{}'; errs+=("meta")
    fi
    if ! releases=$(gh api "repos/$owner/$name/releases?per_page=20" \
            --jq 'map({tag: .tag_name, date: .published_at})' 2>/dev/null); then
        releases='[]'; errs+=("releases")
    fi
    if [ "$(jq 'length' <<<"$releases")" -eq 0 ]; then
        if ! releases=$(gh api "repos/$owner/$name/tags?per_page=20" \
                --jq 'map({tag: .name, date: null})' 2>/dev/null); then
            releases='[]'; errs+=("tags")
        fi
    fi
    errs_json=$(jq -nc --args '$ARGS.positional' "${errs[@]}")

    jq -n --arg owner "$owner" --arg name "$name" --argjson meta "$meta" \
        --argjson releases "$releases" --argjson errs "$errs_json" \
        '{owner: $owner, name: $name} + $meta + {recent_releases: $releases}
         + (if $errs == [] then {} else {github_errors: $errs} end)'
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
process_dep() {
    local idx=$1 path=$2 version=$3 outfile=$4
    local repo_url owner name meta scrutiny dependents vendor_pkgs

    scrutiny=$(scrutiny_flags_json "$version")

    if is_skipped "$path"; then
        jq -n --arg path "$path" --arg version "$version" --argjson scrutiny "$scrutiny" \
            '{path: $path, version: $version, skipped: true,
              skip_reason: "trusted-prefix", extra_scrutiny: $scrutiny}' >"$outfile"
        return
    fi

    repo_url=$(resolve_repo_url "$path" "$version")
    owner=""; name=""
    if [[ "$repo_url" =~ ^https?://github\.com/([^/]+)/([^/]+?)(\.git)?$ ]]; then
        owner="${BASH_REMATCH[1]}"
        name="${BASH_REMATCH[2]}"
    elif [[ "$path" == github.com/*/* ]]; then
        owner=$(printf '%s' "$path" | cut -d/ -f2)
        name=$(printf '%s' "$path" | cut -d/ -f3)
        repo_url=${repo_url:-"https://github.com/$owner/$name"}
    fi

    meta='null'
    if [ -n "$owner" ] && [ -n "$name" ]; then
        meta=$(github_meta "$owner" "$name") || meta='null'
    fi

    dependents=$(count_dependents "$path@$version" "$WORK/graph.txt")
    vendor_pkgs=$(count_vendor_packages "$path")

    jq -n \
        --arg path "$path" \
        --arg version "$version" \
        --argjson scrutiny "$scrutiny" \
        --arg repo_url "$repo_url" \
        --argjson github "$meta" \
        --argjson dependents "$dependents" \
        --argjson vendor_pkgs "$vendor_pkgs" \
        '{
            path: $path,
            version: $version,
            skipped: false,
            extra_scrutiny: $scrutiny,
            repo_url: (if $repo_url == "" then null else $repo_url end),
            github: $github,
            blast_radius: {modules: $dependents, vendor_packages: $vendor_pkgs}
        }' >"$outfile"
}

main() {
    require_cmd go jq gh awk

    local project_dir=${1:-.}
    cd "$project_dir"
    [ -f go.mod ] || die "no go.mod in $(pwd)"

    WORK=$(mktemp -d)
    trap 'rm -rf "$WORK"' EXIT

    log "project: $(go list -m 2>/dev/null || echo unknown)"

    go mod edit -json | jq -r '.Require[]? | select(.Indirect != true) | [.Path, .Version] | @tsv' >"$WORK/deps.tsv"
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
