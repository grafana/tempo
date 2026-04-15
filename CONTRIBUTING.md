# Contributing

Tempo uses GitHub to manage reviews of pull requests:

- If you have a trivial fix or improvement, go ahead and create a pull request.
- If you plan to do something more involved, discuss your ideas on the relevant GitHub issue first.

## Before you begin

Before submitting a pull request, read this guide in full. The PR checklist also requires it.

## AI-assisted contributions

We accept contributions where AI tools (GitHub Copilot, ChatGPT, Claude, etc.) were used during development. However, using AI doesn't lower the bar — it shifts responsibility entirely onto you, the contributor.

### Disclosure

If AI tools generated a significant portion of the code or documentation you're submitting, say so in the PR description. State which tool(s) you used and what you used them for. Trivial uses (autocomplete, grammar checking) don't need disclosure; anything more substantial does.

### You own every line

By submitting a PR you certify the [Developer Certificate of Origin (DCO)](https://developercertificate.org/) — that you have the right to submit the code under this project's license. AI tools cannot certify the DCO. You are the author of record, regardless of how the code was generated.

This means:
- Read and understand every line before submitting. If you can't explain it, don't submit it.
- Autonomous agents submitting PRs without a human reviewing the output are not acceptable.
- Don't use AI to write your PR description or issue comments — those need to accurately represent your understanding of the change.

### Quality and correctness

AI-generated code tends to fail in specific ways. Before submitting, check for:
- **Hallucinated APIs**: functions or methods that don't exist in the libraries used
- **Fake dependencies**: package names that sound plausible but don't exist or aren't imported elsewhere in the codebase — verify every new dependency is real and actively maintained
- **Incorrect edge case handling**: AI often produces code that looks right but handles error paths or boundary conditions wrong
- **Style drift**: AI output often doesn't match the project's conventions — run `make fmt` and `make lint` and fix everything before submitting

All the normal requirements still apply: tests, documentation, changelog entry, passing CI. Unreviewed AI output that wastes maintainer time is not a contribution.

### License and copyright

AI tools may reproduce fragments from their training data. If you're using a tool that can flag similarity to public repositories (such as GitHub Copilot's code referencing), use that feature. If a generated snippet looks like it may have come from an incompatibly-licensed source, rewrite it manually.

## Dependency management

We use [Go modules](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more) to manage dependencies on external packages.
This requires a working Go environment with version 1.26 or greater and git installed.

To add or update a new dependency, use the `go get` command:

```bash
# Pick the latest tagged release.
go get example.com/some/module/pkg

# Pick a specific version.
go get example.com/some/module/pkg@vX.Y.Z
```

Before submitting please run the following to verify that all dependencies and proto definitions are consistent:

```bash
make vendor-check
```

## Project structure

```
cmd/
  tempo/              - main tempo binary
  tempo-cli/          - cli tool for directly inspecting blocks in the backend
  tempo-vulture/      - bird-themed consistency checker.  optional.
  tempo-query/        - jaeger-query GRPC plugin (Apache2 licensed)
docs/
example/              - great place to get started running Tempo
  docker-compose/
  tk/
integration/          - e2e tests
modules/              - top level Tempo components
  backend-worker/
  backend-scheduler/
  distributor/
  overrides/
  querier/
  frontend/
  storage/
opentelemetry-proto/  - git submodule.  necessary for proto vendoring
operations/           - Tempo deployment and monitoring resources (Apache2 licensed)
  jsonnet/
  tempo-mixin/
pkg/
  tempopb/            - proto for interacting with various Tempo services (Apache2 licensed)
tempodb/              - object storage key/value database
vendor/
```

## Coding standards

### Go imports

Imports should follow `std libs`, `external libs`, and `local packages` format:

```go
import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/validation"
)
```

### Error handling

Follow standard Go error handling patterns. Return errors up the call stack; don't swallow them. Avoid `panic` except in truly unrecoverable situations.

**Handle errors immediately and return early:**

Check for errors right after the call and return early. The happy path continues at the normal indentation level — don't nest it in an `else` block.

```go
// good
err := doSomething()
if err != nil {
    level.Error(logger).Log("msg", "failed to do something", "err", err)
    return err
}
// happy path continues here at normal indentation

// avoid
err := doSomething()
if err == nil {
    // happy path buried inside else
} else {
    return err
}
```

**Wrap errors with context using `%w`:**

```go
return fmt.Errorf("failed to create tempodb: %w", err)
```

Always add a short context string that describes what the current function was trying to do. This builds a readable error chain without losing the original error for later inspection.

**Define sentinel errors at the package level:**

```go
var (
    ErrDoesNotExist  = errors.New("does not exist")
    ErrEmptyTenantID = errors.New("empty tenant id")
)
```

Use `errors.New` for sentinel values. Export them (`Err*`) when callers outside the package need to check for them.

**Check errors with `errors.Is` and `errors.As`:**

```go
// Checking sentinel errors
if errors.Is(err, backend.ErrDoesNotExist) { ... }

// Checking for context cancellation
if errors.Is(err, context.Canceled) { ... }

// Extracting a custom error type
var parseErr *ParseError
if errors.As(err, &parseErr) { ... }
```

Prefer `errors.Is` and `errors.As` over direct equality checks or string matching — they traverse the error chain created by `%w` wrapping.

**Define custom error types for structured error data:**

When callers need to inspect error fields (not just identity), define a struct that implements the `error` interface. Add an `Unwrap()` method if the type wraps another error.

```go
type ParseError struct {
    msg  string
    line int
    col  int
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("parse error at line %d, col %d: %s", e.line, e.col, e.msg)
}
```

**Log errors with structured key-value pairs:**

```go
level.Error(logger).Log("msg", "failed to flush block", "tenant", tenantID, "err", err)
level.Warn(logger).Log("msg", "skipped span processing", "err", err)
```

Use `level.Error` for unexpected failures and `level.Warn` for expected or recoverable conditions. Always include `"err", err` as the last key-value pair. Use rate-limited loggers (`log.NewRateLimitedLogger`) for errors that can fire at high frequency.

### `defer`

Use `defer` to pair cleanup with acquisition — the cleanup statement should appear immediately after the call that requires it, not at the end of the function.

**Close iterators and readers immediately after opening:**

```go
iter, err := block.Iterator()
if err != nil {
    return err
}
defer iter.Close()
```

**Unlock mutexes immediately after locking:**

```go
mu.Lock()
defer mu.Unlock()
```

**Cancel contexts immediately after creating them:**

```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

**End spans immediately after starting them:**

```go
ctx, span := tracer.Start(ctx, "operationName")
defer span.End()
```

**Stop tickers and timers immediately after creating them:**

```go
ticker := time.NewTicker(interval)
defer ticker.Stop()
```

**Use anonymous `defer` functions for conditional or error-sensitive cleanup:**

When cleanup logic depends on the function's return value or needs to check errors, use an anonymous function. A common pattern is to record span errors only on failure:

```go
var err error
defer func() {
    if err != nil {
        span.RecordError(err)
    }
}()
```

Another common use is recovering from panics in long-running goroutines:

```go
defer func() {
    if r := recover(); r != nil {
        level.Error(logger).Log("msg", "recovered from panic", "err", r, "stack", string(debug.Stack()))
        err = errors.New("recovered from panic")
    }
}()
```

And for graceful shutdown — stopping subservices only when startup fails mid-way:

```go
defer func() {
    if err != nil && w.subservices != nil {
        if stopErr := services.StopManagerAndAwaitStopped(context.Background(), w.subservices); stopErr != nil {
            level.Error(logger).Log("msg", "failed to stop dependencies", "err", stopErr)
        }
    }
}()
```

### Interface implementation

Verify interface implementation at compile time at the top of files:

```go
var _ SomeInterface = (*ConcreteType)(nil)
```

### Instrumentation

Every non-trivial component should emit metrics, logs, and traces. When adding new features, include observability from the start — don't add it as an afterthought.

- **Metrics**: Tempo is instrumented with [Prometheus metrics](https://prometheus.io/). It emits RED metrics (rate, errors, duration) for most services and backends. Use `promauto` for automatic registration. Name metrics as `tempo_component_metric_unit`, for example `tempo_distributor_ingester_appends_total`. The relevant dashboards are in the [Tempo mixin](operations/tempo-mixin).
- **Logs**: Tempo uses the [go-kit log](https://pkg.go.dev/github.com/go-kit/log) library and emits structured logs in `key=value` (logfmt) format. Use level functions from `github.com/go-kit/log/level`, for example `level.Info(logger).Log("msg", "started", "tenant", tenantID)`. Use rate-limited logging for high-frequency events.
- **Traces**: Tempo uses [OpenTelemetry](https://pkg.go.dev/go.opentelemetry.io) for tracing instrumentation.

### Testing

We try to ensure that most functionality of Tempo is well tested.

- **Unit tests**: Test the functionality of code in isolation. Place `*_test.go` files alongside the code they test. Prefer table-driven tests with `t.Run()` subtests for scenarios with multiple inputs.
- **Local testing**: Use the [examples provided](example) with `docker-compose`, `tanka`, or `helm` to set up a local deployment and test newly added functionality.
- **Integration tests**: Test the ingest and query path end-to-end. These live in the [integration/e2e](integration/e2e) folder.

Use the [testify](https://github.com/stretchr/testify) library for assertions (`assert` for non-fatal checks, `require` for fatal ones).

Run tests before submitting:

```bash
make test
```

For coverage information:

```bash
make test-with-cover
```

A CI job runs these tests on every PR.

### Linting

Run linting before submitting your PR:

```bash
make lint
```

To lint only the changes relative to a base branch (faster for large PRs):

```bash
make lint base=main
```

Fix formatting issues with:

```bash
make fmt
```

This requires `gofumpt` and `goimports` to be in your `$PATH`. This project uses `gofumpt` (a stricter superset of `gofmt`) for formatting. Configure your editor using the [gofumpt documentation](https://github.com/mvdan/gofumpt), or run `make fmt` before committing.

If you changed any jsonnet or libsonnet files, also run:

```bash
make jsonnetfmt
```

This requires the `jsonnetfmt` binary in your `$PATH`.

### Compiling jsonnet

To compile jsonnet files, run:

```bash
make jsonnet
```

This requires `jsonnet`, `jsonnet-bundler`, and `tanka` binaries in your `$PATH`.

### Code generation

If you change any `.proto` files, regenerate the Go code:

```bash
make gen-proto
```

If you change the TraceQL grammar (`.y` files), regenerate the parser:

```bash
make gen-traceql
```

Generated files (`*.pb.go`, `*.y.go`, `*.gen.go`) are excluded from formatting and linting — don't edit them by hand.

## Pull requests

### Description

Every PR must have a clear description covering:

- **What this PR does**: Summarize the change and the motivation behind it.
- **Which issue(s) this PR fixes**: Use `Fixes #<issue number>` so the issue closes automatically on merge.

### Checklist

Before marking a PR ready for review, confirm:

- [ ] Tests updated or added for the changed behavior
- [ ] Documentation added or updated (refer to the [Documentation](#documentation) section)
- [ ] `CHANGELOG.md` updated (refer to the [Changelog entries](#changelog-entries) section)

### Commit messages

Use a semantic prefix in commit messages:

```
<type>: <short description>
```

Common types:

- `fix:` — bug fix
- `feat:` — new feature
- `enhancement:` — improvement to an existing feature
- `chore:` — maintenance, dependency updates, build changes
- `refactor:` — code restructuring with no behavior change
- `docs:` — documentation only

Keep the subject line concise and lowercase after the prefix. Examples:

```
fix: use counter instead of gauge for compactor deduped spans metric
enhancement: deduplicate spans within traces during block builder
chore(deps): update module google.golang.org/api to v0.267.0
```

### Changelog entries

All PRs that change behavior (features, enhancements, bug fixes, breaking changes) must include a `CHANGELOG.md` entry. Dependency-only updates and pure internal refactors do not require one.

Add your entry under `## main / unreleased` in this format:

```
* [TYPE] Short description of the change [#<PR>](https://github.com/grafana/tempo/pull/<PR>) (@your-github-handle)
```

Valid types in order of precedence: `[CHANGE]`, `[FEATURE]`, `[ENHANCEMENT]`, `[BUGFIX]`.

Mark breaking changes explicitly:

```
* [CHANGE] **BREAKING CHANGE** Description of what changed and what users must do [#<PR>](https://github.com/grafana/tempo/pull/<PR>) (@your-github-handle)
```

Keep entries ordered as `[CHANGE]`, `[FEATURE]`, `[ENHANCEMENT]`, `[BUGFIX]` within each release section.

### Keeping your PR up to date

Rebase your PR on `main` if it gets out of sync. Don't merge `main` into your branch.

## Documentation

Anyone can help with Tempo's documentation by writing new content, updating existing content, or creating an issue.
Current documentation projects are tracked in GitHub issues. Browsing through issues is a good way to find something to work on.

### Directory structure

Tempo documentation is located in the `docs` directory. The `docs` directory has three folders:

- `design-proposals`: Used for project and feature proposals. This content is not published with the product documentation.
- `internal`: Used for internal process-related content, including diagrams.
- `sources`: All of the product documentation resides here.
  - The `helm-charts` folder contains the documentation for the `tempo-distributed` Helm chart, https://grafana.com/docs/helm-charts/tempo-distributed/next/
  - The `tempo` folder contains the product documentation, https://grafana.com/docs/tempo/latest/

### Contribute to documentation

Once you know what you would like to write, use the [Writer's Toolkit](https://grafana.com/docs/writers-toolkit/writing-guide/contribute-documentation/) for information on creating good documentation.
The toolkit also provides [document templates](https://github.com/grafana/writers-toolkit/tree/main/docs/static/templates) to help get started.

When you create a PR for documentation, add the `type/doc` label to identify the PR as contributing documentation.

If your content needs to be added to a previous release, use the `backport` label for the version. When your PR is merged, the backport label triggers an automatic process to create an additional PR to merge the content into the version's branch. Check the PR for content that might not be appropriate for the version. For example, if you fix a broken link on a page and then backport to Tempo 1.5, you would not want any TraceQL information to appear.

### Preview documentation

To preview the documentation locally, run `make docs` from the root folder of the Tempo repository. This uses
the `grafana/docs` image which internally uses Hugo to generate the static site. The site is available on `localhost:3002/docs/`.

> **Note** The `make docs` command uses a lot of memory. If it is crashing, make sure to increase the memory allocated to Docker
> and try again.

### Publishing process

Tempo uses a CI action to sync documentation to the [Grafana website](https://grafana.com/docs/tempo/latest). The CI is
triggered on every merge to main in the `docs` subfolder.

The `helm-charts` folder is published from Tempo's next branch. The Tempo documentation is published from the `latest` branch.

## Debugging

Using a debugger can be useful to find errors in Tempo code. This [example](./example/docker-compose/debug)
shows how to debug Tempo inside docker-compose.
