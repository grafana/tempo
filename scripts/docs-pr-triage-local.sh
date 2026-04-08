#!/usr/bin/env bash
# Local docs PR triage: merged PRs on grafana/tempo → heuristic classification (docs-pr-check–style)
# → optional GitHub issue on YOUR fork. Use with cron/launchd Mon/Wed — NOT GitHub Actions.
#
# Does not run the interactive /docs-pr-check skill; see scripts/docs-pr-classify.jq for rules.
#
# Usage:
#   ./scripts/docs-pr-triage-local.sh              # create issue (gh auth; ISSUE_REPO optional)
#   ./scripts/docs-pr-triage-local.sh --dry-run   # markdown only
#
# Env: UPSTREAM_REPO (default grafana/tempo), SINCE_DAYS (default 7), ISSUE_REPO, NO_ISSUE=1
#     DOCS_TEMPO_ROOT — optional path to shipped docs tree (default: repo ../docs/sources/tempo).
#       When set and `rg` is available, keyword search approximates /docs-pr-check step “search docs/sources/tempo”.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CLASSIFY_JQ="${SCRIPT_DIR}/docs-pr-classify.jq"
DOCS_TEMPO_ROOT="${DOCS_TEMPO_ROOT:-$REPO_ROOT/docs/sources/tempo}"

# Optional: same-file keyword hits in local docs (two-keyword match upgrades “Docs needed” → “Docs update needed”).
# Returns JSON object or literal null.
doc_local_hints_json() {
	local title="$1"
	if [[ ! -d "$DOCS_TEMPO_ROOT" ]] || ! command -v rg >/dev/null 2>&1; then
		printf '%s' 'null'
		return 0
	fi

	local cleaned
	cleaned=$(echo "$title" | sed -E 's/^\[[^]]+\]\s*//; s/^(feat|fix|chore|docs)(\([^)]+\))?:\s*//i' | tr '[:upper:]' '[:lower:]')
	local words
	words=$(echo "$cleaned" | tr -cs 'a-z0-9' '\n' | awk 'length >= 5' | awk '!/^(the|and|with|from|for|this|that|into|over|when|were|after|before|there|which|while|where|would|could|about|remove|update|fixed|merge|local|store|trace|traces|error|using|added|based|being|tests|test|minor|fix|chore|docs|opt-in|optin)$/ {print}' | head -3)

	if [[ -z "$words" ]]; then
		printf '%s' 'null'
		return 0
	fi

	local -a arr=()
	while IFS= read -r line; do
		[[ -n "$line" ]] && arr+=("$line")
	done <<<"$words"

	local n=${#arr[@]}
	local kw0="${arr[0]:-}"
	local kw1="${arr[1]:-}"

	local two_files='[]'
	if [[ $n -ge 2 && -n "$kw0" && -n "$kw1" ]]; then
		local f
		while IFS= read -r f; do
			[[ -z "$f" || ! -f "$f" ]] && continue
			if rg -q -i -F "$kw1" "$f" 2>/dev/null; then
				two_files=$(jq -n --argjson arr "$two_files" --arg f "$f" '$arr + [$f] | unique')
			fi
		done < <(rg -l -i -F "$kw0" --glob '*.md' "$DOCS_TEMPO_ROOT" 2>/dev/null || true)
	fi

	local count
	count=$(echo "$two_files" | jq 'length')
	if [[ "${count:-0}" -gt 0 ]]; then
		local keywords_json
		keywords_json=$(printf '%s\n' "${arr[@]}" | jq -R -s -c 'split("\n") | map(select(length > 0))')
		jq -n \
			--argjson keywords "$keywords_json" \
			--argjson files "$two_files" \
			--arg kind "two_keyword" \
			'{keywords: $keywords, files: $files, match_kind: $kind}'
		return 0
	fi

	if [[ -n "$kw0" ]]; then
		local one_files
		one_files=$(rg -l -i -F "$kw0" --glob '*.md' "$DOCS_TEMPO_ROOT" 2>/dev/null | head -8 | jq -R -s -c 'split("\n") | map(select(length>0))')
		local oc
		oc=$(echo "$one_files" | jq 'length')
		if [[ "${oc:-0}" -gt 0 ]]; then
			jq -n \
				--argjson keywords "$(printf '%s\n' "$kw0" | jq -R -s -c 'split("\n") | map(select(length>0))')" \
				--argjson files "$one_files" \
				--arg kind "single" \
				'{keywords: $keywords, files: $files, match_kind: $kind}'
			return 0
		fi
	fi

	printf '%s' 'null'
}

UPSTREAM="${UPSTREAM_REPO:-grafana/tempo}"
RAW_DAYS="${SINCE_DAYS:-7}"
RAW_DAYS="${RAW_DAYS//[[:space:]]/}"
if [[ -z "$RAW_DAYS" || ! "$RAW_DAYS" =~ ^[0-9]+$ ]]; then
	DAYS=7
else
	DAYS="$RAW_DAYS"
fi

DRY=0
[[ "${1:-}" == "--dry-run" ]] && DRY=1
[[ "${NO_ISSUE:-}" == "1" ]] && DRY=1

if [[ "$(uname -s)" == "Darwin" ]]; then
	SINCE=$(date -u -v-${DAYS}d +%Y-%m-%d)
else
	SINCE=$(date -u -d "${DAYS} days ago" +%Y-%m-%d)
fi
CUTOFF_ISO="${SINCE}T00:00:00Z"
RUN_TS=$(date -u +"%Y-%m-%d %H:%M UTC")

SEARCH_Q="repo:${UPSTREAM} is:pr is:merged merged:>=${SINCE}"
ENCODED_Q=$(printf '%s' "$SEARCH_Q" | jq -sRr @uri)

curl_github() {
	local url="$1"
	local use_auth="${2:-auth}"
	local args=(
		-sS
		-H "Accept: application/vnd.github+json"
		-H "X-GitHub-Api-Version: 2022-11-28"
		-H "User-Agent: tempo-doc-work-docs-pr-triage-local"
	)
	if [[ "$use_auth" != "noauth" && -n "${GITHUB_TOKEN:-}" ]]; then
		args+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
	fi
	curl "${args[@]}" "$url"
}

fetch_merged_list() {
	local page=1
	local merged='[]'
	while [[ "$page" -le 15 ]]; do
		local resp
		resp="$(curl_github "https://api.github.com/search/issues?q=${ENCODED_Q}&per_page=100&page=${page}" auth)"
		if ! echo "$resp" | jq -e . >/dev/null 2>&1; then
			resp="$(curl_github "https://api.github.com/search/issues?q=${ENCODED_Q}&per_page=100&page=${page}" noauth)"
		fi
		if echo "$resp" | jq -e 'has("message") and (.message | type == "string") and (.message | length > 0)' >/dev/null 2>&1; then
			echo "Search API: $(echo "$resp" | jq -r '.message')" >&2
			exit 1
		fi
		local chunk n total
		chunk=$(echo "$resp" | jq '[(.items // [])[] | {number, title, url: .html_url}]')
		n=$(echo "$chunk" | jq 'length')
		[[ "$n" -eq 0 ]] && break
		merged=$(echo "$merged" "$chunk" | jq -s '.[0] + .[1]')
		total=$(echo "$resp" | jq '.total_count // 0')
		if [[ $((page * 100)) -ge "$total" ]] || [[ "$n" -lt 100 ]]; then
			break
		fi
		page=$((page + 1))
	done

	local cnt
	cnt=$(echo "$merged" | jq 'length')
	if [[ "$cnt" -eq 0 ]]; then
		merged='[]'
		page=1
		while [[ "$page" -le 25 ]]; do
			local url="https://api.github.com/repos/${UPSTREAM}/pulls?state=closed&sort=updated&direction=desc&per_page=100&page=${page}"
			local prs
			prs="$(curl_github "$url" auth)"
			if echo "$prs" | jq -e 'type == "object" and (.message != null)' >/dev/null 2>&1; then
				prs="$(curl_github "$url" noauth)"
			fi
			chunk=$(echo "$prs" | jq --arg c "$CUTOFF_ISO" '[.[] | select(.merged_at != null) | select(.merged_at >= $c) | {number, title, url: .html_url}]')
			n=$(echo "$chunk" | jq 'length')
			[[ "$n" -gt 0 ]] && merged=$(echo "$merged" "$chunk" | jq -s '.[0] + .[1]')
			local plen
			plen=$(echo "$prs" | jq 'length')
			[[ "$plen" -lt 100 ]] && break
			page=$((page + 1))
		done
	fi
	echo "$merged" | jq 'unique_by(.number) | sort_by(.number) | reverse'
}

LIST_JSON="$(fetch_merged_list)"
COUNT=$(echo "$LIST_JSON" | jq 'length')

if [[ "$COUNT" -eq 0 ]]; then
	echo "No merged PRs in ${UPSTREAM} since ${SINCE} (${DAYS} days)." >&2
	exit 0
fi

ROWS='[]'
while read -r num; do
	[[ -z "$num" ]] && continue
	pjson=$(gh pr view "$num" --repo "$UPSTREAM" --json number,title,body,files,labels 2>/dev/null || echo '')
	[[ -z "$pjson" ]] && continue
	url="https://github.com/${UPSTREAM}/pull/${num}"
	title_for_search=$(echo "$pjson" | jq -r '.title')
	hints=$(doc_local_hints_json "$title_for_search")
	class=$(echo "$pjson" | jq --argjson h "$hints" '. + {doc_local_hints: $h}' | jq -f "$CLASSIFY_JQ")
	row=$(echo "$pjson" | jq -c --argjson c "$class" --arg url "$url" '
		$c + {number: .number, title: .title, url: $url}
	')
	ROWS=$(jq -n --argjson arr "$ROWS" --argjson r "$row" '$arr + [$r]')
done < <(echo "$LIST_JSON" | jq -r '.[].number')

BODY_FILE=$(mktemp)
trap 'rm -f "$BODY_FILE"' EXIT

DOCS_NEEDED=$(echo "$ROWS" | jq '[.[] | select(.classification == "Docs needed")] | sort_by(.priority == "high" | not)')
DOCS_UPDATE=$(echo "$ROWS" | jq '[.[] | select(.classification == "Docs update needed")]')
PR_LIST_NEEDED=$(echo "$ROWS" | jq -r '[.[] | select(.classification == "Docs needed" or .classification == "Docs update needed") | .number] | map("#" + tostring) | join(", ")')

{
	echo "## Docs triage report (local heuristic)"
	echo ""
	echo "| | |"
	echo "|--|--|"
	echo "| **When** | ${RUN_TS} |"
	echo "| **Window** | Merged in \`${UPSTREAM}\` on or after **${SINCE}** (**${DAYS}**-day lookback) |"
	echo "| **PRs in window** | **${COUNT}** |"
	echo "| **Rules** | \`scripts/docs-pr-classify.jq\` + optional local doc keyword search (\`DOCS_TEMPO_ROOT\`) — not a substitute for interactive \`/docs-pr-check\` |"
	echo ""
	echo "### Summary by classification"
	echo ""
	echo "| Classification | Count |"
	echo "|----------------|-------|"
	echo "$ROWS" | jq -r '
		group_by(.classification)
		| sort_by(.[0].classification)
		| .[]
		| "| \(.[0].classification) | \(length) |"
	'
	echo ""
	echo "### Full table"
	echo ""
	echo "| PR | Title | Classification | Priority | Notes |"
	echo "|----|-------|----------------|----------|-------|"
	echo "$ROWS" | jq -r 'sort_by(.number) | reverse | .[] |
		"| [#\(.number)](\(.url)) | \(.title | gsub("\\|"; "/")) | \(.classification) | \(.priority) | \(.notes | gsub("\\|"; "/")) |"'
	echo ""
	echo "### Gap summary (prioritized)"
	echo ""
} >"$BODY_FILE"

echo "$DOCS_NEEDED" | jq -r '
	if length == 0 then
		"**Docs needed** — none flagged by heuristics.\n"
	else
		"**Docs needed** (\(length)) — address first:\n\n" +
		"| PR | Title | Notes |\n|----|-------|-------|\n" +
		(map("| [#\(.number)](\(.url)) | \(.title | gsub("\\|"; "/")) | \(.notes | gsub("\\|"; "/")) |") | join("\n")) +
		"\n"
	end
' >>"$BODY_FILE"

echo "$DOCS_UPDATE" | jq -r '
	if length == 0 then
		"**Docs update needed** — none flagged.\n"
	else
		"**Docs update needed** (\(length)) — verify shipped docs match behavior:\n\n" +
		"| PR | Title | Notes |\n|----|-------|-------|\n" +
		(map("| [#\(.number)](\(.url)) | \(.title | gsub("\\|"; "/")) | \(.notes | gsub("\\|"; "/")) |") | join("\n")) +
		"\n"
	end
' >>"$BODY_FILE"

{
	echo "### Copy-paste for Cursor"
	echo ""
	if [[ -n "$PR_LIST_NEEDED" && "$PR_LIST_NEEDED" != "" ]]; then
		echo "Run **\`/docs-pr-check\`** with:"
		echo ""
		echo '```text'
		echo "PRs on grafana/tempo: ${PR_LIST_NEEDED}"
		echo '```'
	else
		echo "No PRs in **Docs needed** / **Docs update needed** buckets from heuristics. Optionally run **\`/docs-pr-check\`** on any row above you are unsure about."
	fi
	echo ""
	echo "### Next steps"
	echo ""
	echo "1. Use **\`/docs-pr-check\`** in this repo for semantic review (search \`docs/sources/tempo\`, checklist nuance)."
	echo "2. Use **\`/docs-pr-write\`** / **\`/docs-review\`** for follow-up work."
	echo "3. Tune **\`scripts/docs-pr-classify.jq\`** if a row is repeatedly wrong."
} >>"$BODY_FILE"

cat "$BODY_FILE"

if [[ "$DRY" -eq 1 ]]; then
	exit 0
fi

ISSUE_REPO="${ISSUE_REPO:-}"
if [[ -z "$ISSUE_REPO" ]]; then
	ISSUE_REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)
fi
if [[ -z "$ISSUE_REPO" ]]; then
	echo "Set ISSUE_REPO=owner/repo or run from a git clone with \`gh\` auth." >&2
	exit 1
fi

gh label create docs-triage --color FBCA04 --description "Heuristic docs triage (local cron)" --repo "$ISSUE_REPO" 2>/dev/null || true

TITLE="Docs triage: ${UPSTREAM} (${SINCE}–, ${DAYS}d heuristic)"
gh issue create --repo "$ISSUE_REPO" --title "$TITLE" --body-file "$BODY_FILE" --label docs-triage
echo "Created issue on ${ISSUE_REPO}" >&2
