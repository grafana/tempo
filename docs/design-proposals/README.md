# Design Proposals

This directory contains design proposals to modify or extend Grafana Tempo.

Everyone is allowed and encouraged to create new design proposals.
A design proposal is especially recommended for:

- Controversial changes
- Large new features
- Changes with a wide impact

Smaller changes can still be suggested using [the feature request template](https://github.com/grafana/tempo/issues/new?template=feature_request.md).

## How to create a new design proposal

We strive to establish a lightweight process: it should be easy to propose a change and facilitate a discussion. The process should help guide this discussion and ensure document are well archived, but it should not hinder it in any way.

If you are considering starting a design proposal but aren't sure yet if it will be useful, feel free to open a GitHub issue to start a discussion or leave a message in the #tempo Slack channel.

Process:

- Create a new document in this directory
  - The new document should be named `<year>-<month> <title>`. For example `2022-01 Metrics-generator`
  - Please add at least the following front-matter:

```markdown
---
Author: <name> (@<github handle>)
Created: <date this document was created>
Last updated: <date this document was last updated>
---

# <title>

...
```  

- Once the document is ready to be shared, create a Pull Request.
