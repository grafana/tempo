# Contributing

Tempo uses GitHub to manage reviews of pull requests:

- If you have a trivial fix or improvement, go ahead and create a pull request.
- If you plan to do something more involved, discuss your ideas on the relevant GitHub issue.

## Dependency management

We use [Go modules](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more) to manage dependencies on external packages.
This requires a working Go environment with version 1.17 or greater and git installed.

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

# Project Structure

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

## Documentation

Tempo uses a CI action to sync documentation to the [Grafana website](https://grafana.com/docs/tempo/latest). The CI is
triggered on every merge to main in the `docs` subfolder.

To get a preview of the documentation locally, run `make docs` from the root folder of the Tempo repository. This uses
the `grafana/docs` image which internally uses Hugo to generate the static site.

> Note that `make docs` uses a lot of memory and so if its crashing make sure to increase the memory allocated to Docker
and try again.
