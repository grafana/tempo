# Generate context.yaml

Generate `docs/docs-assistant/context.yaml` — the coverage plan that tells the docs update agent what each article must cover. Run this locally to create or fully regenerate the file.

> **Note:** The GitHub Actions docs update workflow handles incremental updates to `docs/docs-assistant/context.yaml` automatically when PRs change user-facing behaviour. Run this prompt locally only when you need to create the file from scratch or bring it back in sync with the current docs structure after major changes.

## Instructions

1. Read `docs/docs-assistant/prompts/docs-assistant.md` to understand the product and documentation conventions
2. Scan `docs/sources/` **recursively** — traverse all subdirectories to find every Markdown file
3. For each file, parse the YAML frontmatter to extract `title`, `weight`, and `description`
4. Follow the rules in `docs/docs-assistant/prompts/update-context.md` to write the file
5. Write the result to `docs/docs-assistant/context.yaml`
