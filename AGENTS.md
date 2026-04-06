# Tempo — Agent Guidance

## Coding Standards

Before writing or modifying Go code, read:

```
Read(file_path: ".agents/guidance/coding.md")
```

Key rules: TDD mandatory (tests first), always wrap errors with `%w`, never silence errors, goroutines must have a defined lifetime, defer resource cleanup, cyclomatic complexity < 15.

## Code Review Standards

Before reviewing code, read:

```
Read(file_path: ".agents/guidance/code-review.md")
```

Work through all 10 passes: security, bug diagnosis, error handling, code quality, performance, Go idioms, architecture, documentation, comment accuracy, reference integrity. Use severity levels (CRITICAL/HIGH/MEDIUM/LOW) to determine whether issues must block merging or can be addressed as targeted fixes.
