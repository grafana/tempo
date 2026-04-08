# Update context.yaml

Rules for writing and updating `docs/docs-assistant/context.yaml` — the coverage plan that tells the docs update agent what each article must cover.

## Structure

```yaml
section_name:
  article-slug:
    - Key topic or concept
    - Another key topic
```

## Rules

- Group articles by directory (section name)
- Order entries within each section by article `weight` frontmatter (ascending)
- Use the filename without `.md` as the slug (for example, `get-started` not `get-started.md`)
- Include `_index.md` section entries using `_index` as the slug
- Write 2–4 concise bullet points per article as plain noun phrases or topics — no leading verbs like "Cover" or "Explain"
- Focus on user-facing behaviour, setup, configuration, and key concepts
- Do not describe internal implementation or development workflows

## Example

```yaml
guides:
  _index:
    - Available guides overview
  onboarding:
    - Using the product to learn core concepts
    - Asking for guided help and best practices
  querying:
    - Generating, explaining, and refining queries
    - Supported data sources and optimization guidance
```
