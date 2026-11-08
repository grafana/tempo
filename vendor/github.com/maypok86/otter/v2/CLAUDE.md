@AGENTS.md

## Claude Code

- Audit skills (`/audit-*`) run in a forked context with the `auditor` agent
  (`.claude/agents/auditor.md`). They are invoked manually only.
- Use `general-purpose` agents, not `auditor`, to verify or consolidate existing reports:
  the auditor always writes its own report and could overwrite the one being checked.
- `/review-change` reviews a diff with roles written for otter (blind concurrency, design
  invariants, regression patterns, contract, tests, performance, simplicity). It is invoked
  manually, like the audits.
