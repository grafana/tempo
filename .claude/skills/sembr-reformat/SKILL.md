---
name: sembr-reformat
description: Reformat prose using Semantic Line Breaks (SemBr) from https://sembr.org while preserving rendered output and meaning. Use when asked to reflow plain text or compatible markup (Markdown, AsciiDoc, reStructuredText, LaTeX, Org, MediaWiki) into semantic one-thought-per-line formatting, or when improving prose diffs and editorial readability.
---

# Reformat With SemBr

Apply Semantic Line Breaks to prose without changing what readers see.
Preserve wording, punctuation, and intent.

## Reformat Workflow

1. Identify prose regions that support soft-wrapped lines.
2. Skip regions where line breaks are syntactically significant:
   - fenced code blocks and indented code blocks
   - tables
   - YAML frontmatter
   - URLs or markup tokens that must stay contiguous
3. Rewrite only line wrapping, not content.
4. Keep paragraphs and list structures intact.
5. Return full rewritten text unless asked for a targeted excerpt.

## SemBr Rules

Apply these rules in order:

1. Break after every sentence ending in `.`, `!`, or `?`.
2. Prefer a break after independent clauses ending in `,`, `;`, `:`, or `—`.
3. Optionally break after dependent clauses when it clarifies structure.
4. Insert a break before an enumerated or itemized list.
5. Optionally add breaks between list items to group related items.
6. Never break inside a hyphenated word.
7. Allow breaks around hyperlinks and before inline markup when helpful.
8. Target 80 columns when possible; allow longer lines for URLs, code spans, or unavoidable markup.

## Quality Checks

Before returning output, verify:

- Rendering-equivalence: no hard-break syntax added accidentally.
- Meaning-equivalence: no word changes or punctuation edits.
- Structure-equivalence: headings, lists, and block boundaries unchanged.
- Diff-friendliness: edits are mostly line-wrap changes.

## Output Style

Use minimal explanation.
Return reformatted text directly.
If any segment cannot be safely reformatted, leave it unchanged and add a one-line note.
