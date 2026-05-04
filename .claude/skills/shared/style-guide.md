## Style guide

Always strictly adhere to the documentation style guide:

Product naming (Grafana organization):

- Write for Grafana Labs users, not staff
- Do not document how to develop on the project
- Do not document deployment for Grafana Cloud products
- Use long product names with "Grafana" in article overviews
- Use short product names without "Grafana" in the article body
- Always use "Grafana Cloud," not "Cloud."
- Mention metrics, logs, traces, profiles in this order

Style:

- Follow Every Page is Page One
- Keep articles short and focused
- Use present simple tense, second-person, and active voice
- Use simple words, short sentences, and few adjectives or adverbs
- Prefer contractions
- Use sentence case for titles, headings, and UI
- Bold UI text
- Reference UI text, not element types such as button or tab

Structure:

- Include a short overview after each heading
- Structure most content under h2 headings
- Use h3 for related subsections
- Do not use lists as a substitute for paragraphs
- Use ", for example," for examples
- Use relative links for internal pages
- End links in `/`, not `.md`
- Use "refer to" instead of "see."
- Separate code and output blocks
- Use `>` quotes for Assistant prompt examples
- Use `<VARIABLE_1>` in code and _VARIABLE_1_ in copy for variables

Where necessary, use the following custom Grafana shortcodes for Hugo:

- If a product or feature is in public preview, use `{{< docs/public-preview product="<PRODUCT|FEATURE">}}`
- Use admonitions sparingly: `{{< admonition type="note|caution|warning" >}}<CONTENT>{{< /admonition >}}`

## Templates

Always include introduction after each heading.
Structure most content under h2 headings.
Use h3 for related subsections.
Use lists sparingly - only for actual list content, not as a substitute for paragraphs.

Determine the topic type from the context and generate the article using the corresponding template. Do not deviate from the template structure.

## Concept template

```markdown
---
title: <Concept name>
menuTitle: <Short title>
weight: <number>
aliases:
  - /old-url/
topicType: concept
versionDate: YYYY-MM-DD
---

# <Concept name>

Introduction explaining the concept, its purpose, and when you use it.

## Why this concept matters

Describe the problem it solves and the outcomes it enables.

## How it works

Explain the underlying ideas, models, or flows.

## Related concepts

Link to related articles.

## Related tasks
```

## Task template

```markdown
---
title: <Task name>
menuTitle: <Short title>
weight: <number>
aliases:
  - /old-url/
topicType: task
versionDate: YYYY-MM-DD
---

# <Task name>

Introduction explaining the goal of the task and when you perform it.

## Before you begin

List requirements.

## <ACTION_VERB_HEADING>

Explain task steps in short sentences.

## Result

Explain how to confirm success.

## Next steps

Link to follow-up tasks.
```

## Reference template

```markdown
---
title: <Reference topic>
menuTitle: <Short title>
weight: <number>
aliases:
  - /old-url/
topicType: reference
versionDate: YYYY-MM-DD
---

# <Reference topic>

Introduction explaining what the reference covers and when to use it.

## Definitions

Define terms.

## Parameters

List fields or settings.

## Examples

Provide minimal examples.

## Related resources

Link to tasks or concepts that use this reference.
```

## Scenario template

```markdown
---
title: <Scenario name>
menuTitle: <Short title>
weight: <number>
aliases:
  - /old-url/
topicType: scenario
versionDate: YYYY-MM-DD
---

# <Scenario name>

Introduction describing the scenario.

## How the scenario works

Explain behavior or flow.

## Goal

## Challenge

## Solution

## Outcome

## Next steps

Link to related articles.
```

## Introduction template

```markdown
---
title: <Product or section name>
menuTitle: <Short title>
weight: <number>
aliases:
  - /old-url/
topicType: introduction
versionDate: YYYY-MM-DD
---

# <Product or section name>

Introduction explaining the purpose of the product, feature, or section.

## How it works at a glance

## Your workflow

## Fundamentals

## Next steps

Link to concept, task, scenario, and reference pages.
```

## Index template

Always use this template for section `_index.md` pages except the home page.

```markdown
---
# Same article frontmatter
---

# Section title

Introduction providing overview of goals and content.

{{< section withDescriptions="true" >}}
```
