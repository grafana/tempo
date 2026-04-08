#!/usr/bin/env bash
# List merged PRs on grafana/tempo and open a triage issue on this repo (tempo-doc-work).
# Used by .github/workflows/docs-upstream-pr-candidates.yml — can also be run locally with
# GITHUB_REPOSITORY=owner/tempo-doc-work and gh auth login.
#
# Upstream list uses the GitHub Search REST API (curl + jq), not `gh pr list`. In Actions,
# `gh pr list --repo other-org/repo` often returns [] (token scope / gh behavior); the Search
# API works anonymously for public repos and matches local `gh` results.

set -euo pipefail

UPSTREAM="${UPSTREAM_REPO:-grafana/tempo}"
DAYS="${SINCE_DAYS:-7}"
LABEL="${ISSUE_LABEL:-docs-triage}"
RUN_DATE=$(date -u +%Y-%m-%d)

if [[ "$(uname -s)" == "Darwin" ]]; then
	SINCE=$(date -u -v-${DAYS}d +%Y-%m-%d)
else
	SINCE=$(date -u -d "${DAYS} days ago" +%Y-%m-%d)
fi

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

{
	echo "Merged pull requests on \`${UPSTREAM}\` since **${SINCE}** (generated **${RUN_DATE}** UTC)."
	echo ""
	echo "Use PR numbers with \`/docs-pr-check\` in your **tempo-doc-work** clone. Not every PR needs docs—skip test-only, vendor, or internal refactors."
	echo ""
	echo "| PR | Title |"
	echo "|----|-------|"
} >"$TMP"

# Search API: merged PRs in repo since date (same semantics as `gh pr list --search merged:>=...`).
SEARCH_Q="repo:${UPSTREAM} is:pr is:merged merged:>=${SINCE}"
ENCODED_Q=$(printf '%s' "$SEARCH_Q" | jq -sRr @uri)

fetch_search_page() {
	local page="$1"
	curl -sS \
		-H "Accept: application/vnd.github+json" \
		-H "X-GitHub-Api-Version: 2022-11-28" \
		-H "User-Agent: tempo-doc-work-upstream-triage" \
		"https://api.github.com/search/issues?q=${ENCODED_Q}&per_page=100&page=${page}"
}

JSON='[]'
PAGE=1
while true; do
	RESP="$(fetch_search_page "$PAGE")"

	if echo "$RESP" | jq -e '.message' >/dev/null 2>&1; then
		echo "GitHub Search API error for ${UPSTREAM}: $(echo "$RESP" | jq -r '.message')" >&2
		exit 1
	fi

	CHUNK=$(echo "$RESP" | jq '[.items[] | {number, title, url: .html_url}]')
	N=$(echo "$CHUNK" | jq 'length')
	if [[ "$N" -eq 0 ]]; then
		break
	fi
	JSON=$(echo "$JSON" "$CHUNK" | jq -s '.[0] + .[1]')

	TOTAL=$(echo "$RESP" | jq '.total_count')
	if [[ $((PAGE * 100)) -ge "$TOTAL" ]] || [[ "$N" -lt 100 ]]; then
		break
	fi
	PAGE=$((PAGE + 1))
	if [[ "$PAGE" -gt 15 ]]; then
		echo "Warning: stopped at page 15 (max 1500 results)." >&2
		break
	fi
done

if ! COUNT="$(echo "$JSON" | jq -e 'length')"; then
	echo "Invalid JSON assembling merged PR list." >&2
	exit 1
fi

if [[ "$COUNT" -eq 0 ]]; then
	{
		echo "No merged PRs in \`${UPSTREAM}\` in the last **${DAYS}** days (since ${SINCE})."
		echo ""
		echo "Increase the window by setting \`SINCE_DAYS\` in the workflow or run \`workflow_dispatch\` after more merges."
	} >"$TMP"
else
	echo "$JSON" | jq -r '.[] | "| [#\(.number)](\(.url)) | \(.title | gsub("\\|"; "/") | gsub("\\n"; " ")) |"' >>"$TMP"
fi

TITLE="Upstream PRs: docs triage — ${RUN_DATE}"

if [[ -z "${GITHUB_REPOSITORY:-}" ]]; then
	echo "--- Preview (no issue created: GITHUB_REPOSITORY unset) ---"
	cat "$TMP"
	exit 0
fi

gh label create "$LABEL" --repo "$GITHUB_REPOSITORY" --color "FBCA04" --description "Upstream Tempo PRs for possible docs work" 2>/dev/null || true

if gh issue create --repo "$GITHUB_REPOSITORY" --title "$TITLE" --body-file "$TMP" --label "$LABEL"; then
	echo "Created issue: ${TITLE}"
else
	echo "gh issue create failed (permissions or API)." >&2
	exit 1
fi
