# Contributing

Tempo uses GitHub to manage reviews of pull requests:

- If you have a trivial fix or improvement, go ahead and create a pull request.
- If you plan to do something more involved, discuss your ideas on the relevant GitHub issue.

## Dependency management is weird.  Please read!

We use [Go modules](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more) to manage dependencies on external packages.
This requires a working Go environment with version 1.15 or greater and git installed.

To add or update a new dependency, use the `go get` command:

```bash
# Pick the latest tagged release.
go get example.com/some/module/pkg

# Pick a specific version.
go get example.com/some/module/pkg@vX.Y.Z
```

When updating dependencies it is important not to run the standard go modules commands due to the way that
OpenTelemetry protos have been vendored.  For now, after making dependency changes run

```bash
make vendor-dependencies
```

We are hoping to improve this in the future.

### Project Structure

```
cmd/
  tempo/              - main tempo binary
  tempo-cli/          - cli tool for directly inspecting blocks in the backend
  tempo-vulture/      - bird-themed consistency checker.  optional.
  tempo-query/        - jaeger-query GRPC plugin
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
  storage/
opentelemetry-proto/  - git submodule.  necessary for proto shenanigans
operations/           - Tempo deployment and monitoring resources
  jsonnet/
  tempo-mixin/
pkg/
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