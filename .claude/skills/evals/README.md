# Running evals for docs-pr skills

This guide explains how to evaluate the `docs-pr-check` and `docs-pr-write` skills using the eval test cases defined in each skill's `evals/evals.json`.

## Workflow context

These two skills cover Phases 1.5 and 1.75 of the Tempo release notes workflow. See the full workflow at `.agents/doc-agents/shared/release-notes-workflow.md`.

```
Phase 1   → Phase 1.5          → Phase 1.75         → Phase 2
Gather      docs-pr-check        docs-pr-write         Draft
PRs         (classify gaps)      (fill gaps)           release notes
```

The evals are written to reflect this context: prompts set up the workflow phase, and assertions check that output is structured to feed cleanly into the next phase.

## Structure

```
.claude/skills/
├── docs-pr-check/
│   └── evals/
│       └── evals.json          # 4 test cases: 3 standalone, 1 workflow (Phase 1.5)
├── docs-pr-write/
│   └── evals/
│       └── evals.json          # 4 test cases: 2 standalone, 2 workflow (Phase 1.75)
└── evals/
    ├── README.md               # this file
    └── evals.json              # 1 integration test (Phase 1.5 → 1.75 handoff)
```

Each evals.json uses a `"mode"` field on each test case to indicate whether it tests standalone or workflow usage. Run both modes when evaluating a skill — they exercise different behaviors.

Results from each iteration go in a separate workspace directory (outside the skills directory):

```
evals-workspace/
└── iteration-1/
    ├── docs-pr-check-eval-1/
    │   ├── with_skill/
    │   │   ├── outputs/
    │   │   ├── timing.json
    │   │   └── grading.json
    │   └── without_skill/
    │       ├── outputs/
    │       ├── timing.json
    │       └── grading.json
    ├── docs-pr-write-eval-1/
    │   └── ...
    └── benchmark.json
```

## Pre-flight: select PRs

Eval prompts use `{PR_N}` placeholders instead of hardcoded PR numbers. Before running any eval, select real PRs to substitute in.

### Why dynamic selection

Hardcoded PR numbers become stale — once a feature is documented or a PR is closed, the fixture no longer tests the skill's general behavior. Selecting PRs fresh each eval run ensures the skill is tested against real, current examples.

### Base filter command

Use this to get a list of candidate PRs, excluding Dependabot and GitHub Actions bot PRs:

```bash
gh pr list --repo grafana/tempo --state merged --limit 100 \
  --json number,title,author,labels,files \
  | jq '[.[] | select(
      .author.login != "dependabot[bot]" and
      .author.login != "github-actions[bot]"
    )] | .[0:20]'
```

Each test case in `evals.json` also has a `pr_selection_command` field with a more targeted filter for its specific `pr_type`. Run that command first; fall back to the base filter if it returns too few results.

### Matching PRs to pr_type

Each test case defines a `pr_type` describing the characteristics required. When reviewing candidates:

- **Docs needed**: PR changed Go, proto, or Helm files; no `docs/` files changed; labeled `type/feature`, `type/enhancement`, or `add to changelog`
- **Docs update needed**: PR changed behavior where existing docs exist but weren't updated; often labeled `type/enhancement`
- **No docs required**: PR is a refactor, test change, dependency bump, or CI update; labeled `type/chore`, `type/ci`, `type/refactor`, or `type/testing`

When in doubt, check the PR body and changed files directly with:

```bash
gh pr view {NUMBER} --repo grafana/tempo --json title,body,files,labels
```

### Rules for selection

- Do not reuse the same PR across multiple test cases in the same eval run
- Do not select PRs authored by bots even if the filter misses them
- For test cases 3 and 4 in `docs-pr-check` and the integration test, select PRs from different areas of the codebase (for example, one from `modules/`, one from `tempodb/`, one from `pkg/`)

### Workflow mode: filling in `{SUGGESTED_TARGET_FILE}` and `{EXISTING_DOCS_FILE}`

Some `docs-pr-write` workflow prompts include a second placeholder for the target file. Run `docs-pr-check` on your selected PR first and use the file path it identifies in the Notes column.

## Running a single eval

For each test case, run it twice — once with the skill, once without.

**With skill:**

```
Execute this task:
- Skill path: .claude/skills/docs-pr-check
- Task: [paste prompt from evals.json]
- Save outputs to: evals-workspace/iteration-1/docs-pr-check-eval-1/with_skill/outputs/
```

**Without skill (baseline):**

```
Execute this task:
- No skill
- Task: [paste same prompt]
- Save outputs to: evals-workspace/iteration-1/docs-pr-check-eval-1/without_skill/outputs/
```

For the integration test (both skills in sequence), provide both skill paths:

```
Execute this task:
- Skills: .claude/skills/docs-pr-check, .claude/skills/docs-pr-write
- Task: [paste integration prompt from evals/evals.json]
- Save outputs to: evals-workspace/iteration-1/integration-eval-1/with_skill/outputs/
```

## Grading

After both runs, grade each assertion from `evals.json` against the actual output. Record results in `grading.json`:

```json
{
  "assertion_results": [
    {
      "text": "Output includes a markdown table with columns: PR, Title, Classification, Notes",
      "passed": true,
      "evidence": "Table present at line 12 with all four required columns"
    }
  ],
  "summary": {
    "passed": 4,
    "failed": 1,
    "total": 5,
    "pass_rate": 0.80
  }
}
```

## Key things to watch for

### docs-pr-check — standalone mode

- Does the agent correctly distinguish user-facing features from internal changes without workflow framing?
- Does it still search `docs/sources/tempo` to check for existing coverage?
- Is the output useful as a direct answer (not just a handoff artifact)?

### docs-pr-check — workflow mode

- Are the four classification categories used consistently?
- Does the gap summary prioritize PRs with no docs over PRs that need updates?
- Is the output formatted so it can be passed to docs-pr-write without re-entry?

### docs-pr-write — standalone mode

- Does the agent identify the right target file without being given one?
- Does it prefer updating existing pages over creating new ones?
- Is the return format still present (files changed, mapping, open items) even when not explicitly requested?

### docs-pr-write — workflow mode

- Does the agent verify field names and defaults against code (not just PR description text)?
- Does it only process `Docs needed` and `Docs update needed` PRs — not `No docs required`?
- Is content concise and action-oriented, structured so Phase 2 can link to it?

### Integration (Phase 1.5 → 1.75)

- Does the agent carry the Phase 1.5 output directly into Phase 1.75 without requiring manual re-entry?
- Are the two phases' outputs clearly labeled in the final response?
- Does the Phase 1.75 output only include PRs that were flagged by Phase 1.5?

## Capturing timing

Record token count and duration for each run in `timing.json`:

```json
{
  "total_tokens": 12400,
  "duration_ms": 18500
}
```

Use these to track whether skill instructions are adding cost without proportionate quality improvement.
