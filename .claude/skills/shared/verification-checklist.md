# Documentation Verification Checklist

> **Intended use:** This file serves two purposes:
> 1. **Human writers** — use as a full reference checklist when writing or reviewing documentation.
> 2. **Agents (docs-pr-write Step 8 only)** — read this file to select the sections relevant to the change type, then present those sections to the user as a handoff checklist. Do not attempt to complete these items autonomously.

Use this checklist when writing or updating documentation to ensure accuracy, consistency, and completeness.

## Pre-Writing

- [ ] Identified user problem or confusion point (GitHub issue, user feedback, etc.)
- [ ] Located relevant codebase files
- [ ] Reviewed existing documentation for context
- [ ] Understood feature scope (component-specific vs. shared)

## Codebase Verification

- [ ] Read implementation code in the source tree (paths from local context, e.g. `modules/` or `pkg/`)
- [ ] Verified feature names match code (check constants, structs, functions)
- [ ] Confirmed configuration struct fields match documentation
- [ ] Checked default values in code (look for the function that registers defaults — path from local context)
- [ ] Verified optional vs. required parameters
- [ ] Understood feature relationships (for example, whether related config options are alternatives or complementary)
- [ ] Checked for version-specific behavior or availability

## Configuration Reference Check

- [ ] Compared documented options with your project's configuration reference page (path from local context)
- [ ] Verified YAML structure matches reference format
- [ ] Ensured all configuration options are documented
- [ ] Checked for missing options in documentation
- [ ] Verified default values match between code and config reference

## Version Compatibility

- [ ] Checked `CHANGELOG.md` for feature introduction version
- [ ] Verified feature availability in target documentation version
- [ ] Noted any version-specific requirements or breaking changes
- [ ] Confirmed feature availability matches the doc version you are targeting (no mixing incompatible release lines)

## Accuracy Checks

- [ ] Verified counts (e.g., "three metrics" not "two metrics")
- [ ] Confirmed default vs. optional labels/features
- [ ] Checked metric names match code constants
- [ ] Verified label and metric names match code constants (refer to local context for specific identifiers)
- [ ] Confirmed component or processor names match code

## Style Guide Compliance

- [ ] Used "refer to" not "see" for links
- [ ] Internal links end with "/" where appropriate
- [ ] Sentence case for all headings
- [ ] Added introduction sentences after headings
- [ ] Used prose instead of lists for explanations
- [ ] Used ", for example," pattern for examples
- [ ] Admonitions used sparingly and appropriately
- [ ] Active voice preferred over passive

## Example Quality

- [ ] Examples demonstrate actual use cases, not edge cases
- [ ] Examples show intended behavior, not default sanitization
- [ ] Code examples are syntactically correct
- [ ] Examples are tested or verified against code
- [ ] Examples include necessary context

## User Clarity

- [ ] Addressed root cause of user confusion, not just symptoms
- [ ] Added explicit clarifications for non-obvious requirements
- [ ] Explained relationships between related features
- [ ] Provided clear guidance on when to use which feature
- [ ] Included warnings or notes for common mistakes

## Completeness

- [ ] All configuration options documented
- [ ] All related features explained
- [ ] Missing features identified and documented
- [ ] Cross-references added where appropriate
- [ ] Related documentation linked

## Final Review

- [ ] Read documentation as a user would
- [ ] Verified all claims can be traced to codebase
- [ ] Checked for consistency with related documentation
- [ ] Ensured documentation addresses the original problem
- [ ] Confirmed examples work as documented

## Post-Submission

- [ ] Documented learnings in knowledge base
- [ ] Updated domain knowledge if new insights discovered
- [ ] Noted any gaps found for future work
