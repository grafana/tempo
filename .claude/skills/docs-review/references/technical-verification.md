# Technical verification

When the review triage classifies a change as **technical** (new features, config changes, behavioral claims, code examples), follow these steps before completing the style review.

## 1. Identify verifiable claims

Read the changed content and flag every statement that makes a factual claim about code behavior: config option names, default values, numeric limits, syntax examples, API behavior, metric/label names, or version availability.

For the specific lookup locations (which files to check for each claim type), refer to the **Codebase Verification**, **Configuration Reference Check**, **Accuracy Checks**, and **Version Compatibility** sections of `.agents/doc-agents/shared/verification-checklist.md`.

## 2. Prioritize and verify

Not every claim needs a code check. Prioritize by risk level:

- **High risk** — Claims involving specific numbers (defaults, limits, sizes), config option names and YAML paths users will copy-paste, or code examples. Divergence here breaks user workflows. Always verify against code.
- **Medium risk** — Feature descriptions from PR text, version availability claims, behavioral descriptions. Verify when possible; flag for human review if the source is unclear.
- **Low risk** — General descriptions of existing, well-documented features. Spot-check if time allows.

Use this ordering within each level:

1. Anything with a specific number (default values, counts, limits, sizes)
2. New features from a PR where the PR description may oversimplify
3. Platform-dependent behavior (e.g., `math.MinInt` vs `math.MinInt64`)
4. Config option names and YAML paths (typos here break users)
5. Code examples that users will copy-paste

For each high- or medium-risk claim, read the relevant source file and confirm the claim is accurate. If you cannot locate the source, flag it for the human reviewer rather than assuming it is correct.

## 3. Report findings

For each claim verified, briefly note:
- The claim as written in the docs
- What the code actually says (with file path)
- Whether they match or diverge

Present divergences as issues in the main review output (Step 8 of the review). Group them separately from style issues.
