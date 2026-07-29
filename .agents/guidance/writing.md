# Writing Standards

Guidance for prose in this repository:
documentation, READMEs, design docs, and agent guidance.
Apply these when writing new prose or editing existing paragraphs.

---

## Semantic Line Breaks (SemBr)

Write Markdown and other soft-wrapped prose using [Semantic Line Breaks](https://sembr.org):
break source lines at semantic boundaries
instead of hard-wrapping at a fixed column or putting whole paragraphs on one line.
Line breaks within a paragraph don't affect the rendered output,
so this is a source-formatting convention only.

### Why

- Diffs show which sentence or clause changed, not a rewrapped paragraph.
- Reviewers can comment on a single sentence instead of a whole paragraph.
- LLM-assisted writing tends to produce long, dense paragraphs;
  SemBr keeps those edits reviewable.

### How

Use the official SemBr agent skill vendored at
[`.claude/skills/sembr-reformat/SKILL.md`](../../.claude/skills/sembr-reformat/SKILL.md)
(from [sembr/skills](https://github.com/sembr/skills), MIT licensed).
It defines the break rules, the regions to skip,
and the quality checks for rendering- and meaning-equivalence.
Agents that support skills discover it automatically;
humans can read it as the rule reference.

### Scope

- Apply SemBr to new prose and to paragraphs you're already editing.
- Don't mass-reformat otherwise-untouched files —
  that churns diffs, which is exactly what SemBr is meant to avoid.
- Leave alone: code blocks, tables, YAML frontmatter,
  and any markup where line breaks are syntactically significant
  (for example, Markdown hard breaks).
