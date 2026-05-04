# Pre-Commit Checklist

Run these before pushing. They cover the most common local checks; CI may run additional validation (e.g. jsonnetfmt, integration tests).

---

## Formatting

```bash
make fmt
```

Uses `gofumpt` and `goimports`. CI runs `make check-fmt` and fails if the tree is dirty — so run this first.

---

## Linting

```bash
make lint
```

Runs `golangci-lint`. CI lints only the diff against the base branch:

```bash
make lint base=origin/main
```

Run the diff-scoped version locally to match CI exactly when working on a large repo.

---

## Unit Tests

```bash
# All packages
make test

# With race detector and coverage (matches CI splits)
make test-with-cover
```

CI splits tests into four jobs — `pkg`, `tempodb`, `tempodb/wal`, and `others`. You can run individual splits to match:

```bash
make test-with-cover-pkg
make test-with-cover-tempodb
make test-with-cover-tempodb-wal
make test-with-cover-others
```

---

## E2e Tests

E2e tests require Docker. They build a local Tempo image before running.

```bash
# Full suite
make test-e2e

# Individual suites
make test-e2e-api
make test-e2e-operations
make test-e2e-limits
make test-e2e-metrics-generator
make test-e2e-storage
```

E2e tests live in `integration/e2e/`. After a run, clean up Docker-owned test directories:

```bash
make test-e2e-clean
```

---

## Minimum Bar Before Opening a PR

1. `make fmt` — no dirty tree
2. `make lint base=origin/main` — no new lint errors
3. `make test` — all unit tests pass
4. `go test -race ./...` (or `make test-with-cover`) — no races in changed packages
