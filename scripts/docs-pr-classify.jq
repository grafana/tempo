# Input: gh pr view JSON (number, title, body, files, labels?, doc_local_hints?)
#   doc_local_hints: null | {keywords: string[], files: string[], match_kind: "two_keyword"|"single"}
# Output: {classification, notes, priority}
# priority: "high" | "medium" | "low" — for gap ordering (high = address first)
#
# Aligns with /docs-pr-check where possible using checklist, paths, labels, and optional
# local keyword search in docs/sources/tempo (see docs-pr-triage-local.sh).

def has_docs: (.files // []) | map(.path) | any(test("^docs/"; ""));
def doc_checked: (.body // "") | test("- \\[\\s*[xX]\\s*\\]\\s*Documentation added");
def doc_unchecked: (.body // "") | test("- \\[\\s*\\]\\s*Documentation added");

def label_names: (.labels // []) | map(.name);
def label_skip_changelog: label_names | any(test("changelog/skip|skip-changelog|skip_changelog"; "i"));
def label_dependencies: label_names | any(test("^dependencies$"; "i"));

def all_paths: (.files // []) | map(.path);
def only_ci_paths:
  (all_paths | length) > 0 and (all_paths | all(test("^\\.github/")));

def dep_like:
     (.title | test("(?i)^(fix|chore)\\(deps\\)|^chore\\(deps\\)|^fix\\(deps\\)|renovate|lock file|digest to|docker tag|writers-toolkit"))
  or label_dependencies;
def test_only: (.title | test("(?i)^Minor: fix test|flaky test|^fix\\([^)]+\\): .*\\btest\\b"));
def mixin: (.title | test("(?i)\\[tempo-mixin\\]"));
def agent_repo:
  (.title | test("(?i)Claude agent|agent guidance"))
  or ((.files // []) | map(.path) | any(test("\\.agents/|AGENTS\\.md$"; "")));

def title_docs: (.title | test("(?i)\\[DOCS\\]"));

# Performance / internal — no user-facing behavior doc
def perf_internal:
  (.title | test("(?i)Reduce allocations|Reuse Trace|extendReuseSlice|pushBytes|across pushBytes"));

# Strong user-facing signals (config, CLI, architecture, overrides)
def strong_user:
     (.title | test("(?i)\\[vparquet|\\[tempo-cli\\]|per-tenant|override|feat:|feat\\("))
  or (.title | test("(?i)single-binary|distributor.*live-store|ingest path|LocalPush"))
  or (.title | test("(?i)TraceRedactor|ErrTraceHidden|redaction|redact\\b"))
  or (.title | test("(?i)breaking|migration|deprecated"));

# Weaker user signal (still worth /docs-pr-check). Do not use “touched modules/” alone — too many false positives vs /docs-pr-check.
def review_user:
     (.title | test("(?i)^Extend |^Add .*(client|command|API)|new fetch|SearchTagValues"))
  or ((.files // []) | map(.path) | any(test("(^|/)cmd/tempo/|(^|/)docs/"; "")));

# Typical bugfixes / internal fixes — usually release-note or nothing
def bugfix_internal:
     (.title | test("(?i)^fix\\(livestore\\):|^fix\\(traceql\\):"))
  or (.title | test("(?i)SearchTagValues|disk cache|deregister.*ring|intPow|unsuccessful deregister"));

def title_refactor:
  (.title | test("(?i)^(refactor|internal)\\([^)]+\\):"));

# --- main branch (no doc_local_hints yet) ---
def classify_core:
  if title_docs then
    {classification: "Docs present", notes: "Title or work marked [DOCS].", priority: "low"}
  elif has_docs and doc_checked then
    {classification: "Docs present", notes: "Checklist: documentation added + docs/ paths.", priority: "low"}
  elif has_docs and doc_unchecked then
    {classification: "Docs update needed", notes: "docs/ touched but “Documentation added” not checked — verify completeness.", priority: "medium"}
  elif has_docs then
    {classification: "Docs present", notes: "Touches docs/ — spot-check for completeness.", priority: "low"}
  elif only_ci_paths and (has_docs | not) then
    {classification: "No docs required", notes: "CI / workflow only (.github/); no product docs.", priority: "low"}
  elif label_skip_changelog and (strong_user | not) and (review_user | not) and (has_docs | not) then
    {classification: "No docs required", notes: "Label suggests changelog skip (internal / not user-facing).", priority: "low"}
  elif title_refactor and (strong_user | not) and (review_user | not) and (has_docs | not) then
    {classification: "No docs required", notes: "Refactor/internal title; no user-facing doc unless release notes.", priority: "low"}
  elif dep_like or test_only or mixin or agent_repo then
    {classification: "No docs required", notes: (if dep_like then "Dependency / vendor / image" elif test_only then "Tests" elif mixin then "tempo-mixin (ops JSON)" else "Agent / dev-only paths" end), priority: "low"}
  elif perf_internal and (strong_user | not) then
    {classification: "No docs required", notes: "Performance / internal path; no product doc unless release notes call it out.", priority: "low"}
  elif bugfix_internal and (strong_user | not) then
    {classification: "No docs required", notes: "Bugfix; consider one line in release notes if user-visible.", priority: "low"}
  elif strong_user and (has_docs | not) and (doc_checked | not) then
    {classification: "Docs needed", notes: "User-facing change (CLI, config, architecture, or API) without docs in PR — run /docs-pr-check.", priority: "high"}
  elif strong_user and has_docs then
    {classification: "Docs update needed", notes: "User-facing + docs/ — confirm coverage in shipped pages.", priority: "medium"}
  elif review_user and (has_docs | not) and (doc_checked | not) then
    {classification: "Docs needed", notes: "Possible user impact — confirm with /docs-pr-check (search docs/sources/tempo).", priority: "high"}
  elif review_user and has_docs then
    {classification: "Docs update needed", notes: "Touches code + docs — verify narrative matches behavior.", priority: "medium"}
  else
    {classification: "No docs required", notes: "Default: internal/refactor/fix; spot-check if unsure.", priority: "low"}
  end;

def apply_doc_hints($hints):
  if $hints == null then .
  elif .classification == "Docs needed" and ($hints.match_kind == "two_keyword") and (($hints.files | length) > 0) then
    .classification = "Docs update needed"
    | .notes = "Local docs tree matched keywords in the same .md file(s) as /docs-pr-check would search — verify completeness vs PR. Files: "
        + (($hints.files | .[0:4] | join(", "))
        + (if ($hints.files | length) > 4 then "…" else "" end))
    | .priority = "medium"
  elif .classification == "Docs needed" and ($hints.match_kind == "single") and (($hints.files | length) > 0) then
    .notes += " | Hint: keyword appears in shipped docs (may be false positive; confirm)."
  else . end;

(.doc_local_hints // null) as $hints
| del(.doc_local_hints)
| classify_core
| apply_doc_hints($hints)
