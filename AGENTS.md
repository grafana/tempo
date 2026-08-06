# Tempo — Agent Guidance

## Changelog Entries

Never edit `CHANGELOG.md` directly. Every user-facing change adds a YAML entry
under [`.chloggen/`](.chloggen/) instead — read [`.chloggen/README.md`](.chloggen/README.md)
first, then create the entry with `make chlog-new` and validate it with
`make chlog-validate`.

## Coding Standards

Before writing or modifying Go code, read [`.agents/guidance/coding.md`](.agents/guidance/coding.md).

## Code Review Standards

Before reviewing code, read [`.agents/guidance/code-review.md`](.agents/guidance/code-review.md).

## Writing Standards

Before writing or editing Markdown prose (docs, READMEs, design docs), read [`.agents/guidance/writing.md`](.agents/guidance/writing.md).

## Pre-Commit Checklist

Before pushing or opening a PR, read [`.agents/guidance/precommit.md`](.agents/guidance/precommit.md).
