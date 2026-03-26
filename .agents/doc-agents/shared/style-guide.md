## Style guide

Always follow these rules.

- Use long product names with "Grafana" in article overview, short names in body
- Always use "Grafana Cloud," not "Cloud"
- Mention: metrics, logs, traces, profiles (in this order)
- Follow Every Page is Page One approach
- Present simple tense, second-person, active voice
- Simple words, short sentences, few adjectives/adverbs, prefer contractions
- Sentence case for titles, headings, and UI
- Bold UI text, don't reference element types
- Internal links end in "/", use "refer to" not "see"
- Variables: `<VARIABLE_1>` in code, _VARIABLE_1_ in copy
- Separate code and output blocks
- Use ", for example," for examples
- Admonitions sparingly: `{{< admonition type="note|caution|warning" >}}`
- Preview shortcodes: `{{< docs/private-preview >}}` or `{{< docs/public-preview >}}`

## Templates

Always include introduction after each heading.
Structure most content under h2 headings.
Use h3 for related subsections.
Use lists sparingly - only for actual list content, not as a substitute for paragraphs.

Determine the topic type from the context and generate the article using the corresponding template. Do not deviate from the template structure.

Concept template
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

Task Template

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


Reference Template

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

Scenario Template

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

Introduction Template

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

