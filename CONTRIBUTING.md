# Contributing

Tempo uses GitHub to manage reviews of pull requests:

- If you have a trivial fix or improvement, go ahead and create a pull request.
- If you plan to do something more involved, discuss your ideas on the relevant GitHub issue.

## Dependency management

We use [Go modules](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more) to manage dependencies on external packages.
This requires a working Go environment with version 1.18 or greater and git installed.

To add or update a new dependency, use the `go get` command:

```bash
# Pick the latest tagged release.
go get example.com/some/module/pkg

# Pick a specific version.
go get example.com/some/module/pkg@vX.Y.Z
```

Before submitting please run the following to verify that all dependencies and proto defintions are consistent.

```bash
make vendor-check
```

# Project structure

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
  compactor/
  distributor/
  ingester/
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

## Coding Standards

### go imports
imports should follow `std libs`, `externals libs` and `local packages` format

Example
```
import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/validation"
)

```

### Instrumentation

- **Metrics**: Tempo is instrumented with [Prometheus metrics](https://prometheus.io/). It emits RED metrics for most
  services and backends. The relevant dashboards can be found in the [Tempo mixin](operations/tempo-mixin).
- **Logs**: Tempo uses the [go-kit level logging library](https://pkg.go.dev/github.com/go-kit/kit/log/level) and emits
  logs in the `key=value` (logfmt) format.
- **Traces**: Tempo uses the [Jaeger Golang SDK](https://github.com/jaegertracing/jaeger-client-go) for tracing instrumentation.
  As of this writing, only the read path of tempo is instrumented for tracing.

### Testing

We try to ensure that most functionality of Tempo is well tested.

- At the package level, we write unit tests that tests the functionality of the code in isolation.
  These can be found within each package/module as `*_test.go` files.
- Next, a good practice is to use the [examples provided](example) and common tools like `docker-compose`, `tanka` &
  `helm` to set up a local deployment and test the newly added functionality.
- Finally, we write integration tests that often test the functionality of the ingest and query path of Tempo as a
  whole, including the newly introduced functionality. These can be found under the [integration/e2e](integration/e2e)
  folder.

A CI job runs these tests on every PR.

### Debugging

Using a debugger can be useful to find errors in Tempo code. This [example](./example/docker-compose/debug)
shows how to debug Tempo inside docker-compose.

### Linting

Make sure to run

```
make lint
```

before submitting your PR to catch any linting errors. Linting can be fixed using

```
make fmt
```

However, do note that the above command requires the `gofmt` and `goimports` binaries accessible via `$PATH`.

If you have changed any jsonnet or libsonnet files, please run

```
make jsonnetfmt
```

This requires `jsonnetfmt` binary in `$PATH`.

### Compiling jsonnet

To compile jsonnet files please run the following command

```
make jsonnet
```

This requires `jsonnet`, `jsonnet-bundler` and `tanka` binaries in `$PATH`.

## Documentation

Anyone can help with Tempo's documentation by writing new content, updating existing content, or creating an issue.
Current documentation projects are tracked in GitHub issues. Browsing through issues is a good way to find something to work on.

### Directory structure

Tempo documentation is located in the `docs` directory. The `docs` directory has three folders:

- `design-proposals`: Used for project and feature proposals. This content is not published with the product documentation.
- `internal`: Used for internal process-related conteint, including diagrams.
- `sources`: All of the product documentation resides here.
   - The `helm-charts` folder contains the documentation for the `tempo-distributed` Helm chart, https://grafana.com/docs/helm-charts/tempo-distributed/next/
   - The `tempo` folder contains the product documentation, https://grafana.com/docs/tempo/latest/

### Contribute to documentation

Once you know what you would like to write, use the [Writer's Toolkit](https://grafana.com/docs/writers-toolkit/writing-guide/contribute-documentation/) for information on creating good documentation.
The toolkit also provides [document templates](https://github.com/grafana/writers-toolkit/tree/main/docs/static/templates) to help get started.

When you create a PR for documentation, add the `types/doc` label to identify the PR as contributing documentation. 

If your content needs to be added to a previous release, use the `backport` label for the version. When your PR is merged, the backport label triggers an automatic process to create an additional PR to merge the content into the version's branch. Check the PR for content that might not be appropriate for the version. For example, if you fix a broken link on a page and then backport to Tempo 1.5, you would not want any TraceQL information to appear.  

### Preview documentation

To preview the documentation locally, run `make docs` from the root folder of the Tempo repository. This uses
the `grafana/docs` image which internally uses Hugo to generate the static site. The site is available on `localhost:3002/docs/`.

> **Note** The `make docs` command uses a lot of memory. If its crashing make sure to increase the memory allocated to Docker
and try again.

### Publishing process

Tempo uses a CI action to sync documentation to the [Grafana website](https://grafana.com/docs/tempo/latest). The CI is
triggered on every merge to main in the `docs` subfolder.

The `helm-charts` folder is published from Tempo's next branch. The Tempo documentation is published from the `latest` branch.