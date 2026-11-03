#!/usr/bin/env bash
set -euo pipefail

# Exercise the public Go packages outside Tempo's module and vendor directory.
# The local replacement tests this checkout, not public tag/proxy resolution.
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
consumer_dir=$(mktemp -d)
trap 'rm -rf "$consumer_dir"' EXIT
export GOWORK=off
export GOFLAGS=

if [[ "$(cd "$repo_root" && go list -m)" != "github.com/grafana/tempo/v3" ]]; then
    echo 'Tempo must declare the /v3 module path' >&2
    exit 1
fi

cd "$consumer_dir"
go mod init example.com/tempo-consumer
go mod edit -require=github.com/grafana/tempo/v3@v3.0.0
go mod edit "-replace=github.com/grafana/tempo/v3=$repo_root"
cat > consumer_test.go <<'GOEOF'
package consumer

import (
	"testing"

	"github.com/grafana/tempo/v3/pkg/gogocodec"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

func TestConsumer(t *testing.T) {
	if _, err := traceql.Parse(`{ resource.service.name = "checkout" }`); err != nil {
		t.Fatal(err)
	}
	codec := gogocodec.NewCodec()
	request := &tempopb.TraceByIDRequest{TraceID: []byte{1, 2, 3}}
	data, err := codec.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var decoded tempopb.TraceByIDRequest
	if err := codec.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if string(decoded.TraceID) != string(request.TraceID) {
		t.Fatalf("trace ID = %x, want %x", decoded.TraceID, request.TraceID)
	}
}
GOEOF

go mod tidy
go mod graph > /dev/null
# Stale internal imports must not silently pull in a second copy of Tempo.
if go list -m all | grep '^github.com/grafana/tempo ' > /dev/null; then
    echo 'consumer unexpectedly depends on the unversioned Tempo module' >&2
    exit 1
fi
go test -mod=mod ./...
go mod vendor
go test -mod=vendor ./...
