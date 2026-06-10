# Tempo → wiresmith protobuf compiler migration

Tracking document for replacing protoc-gen-gogofaster with
[wiresmith](https://github.com/grafana/wiresmith) for Tempo's own protos.
Wire format is unchanged (standard protobuf, binary-compatible with gogo);
only the generated Go API shape changes.

Toolchain: `wiresmith` CLI (v0.0.0-20260609114539-bee7aa4), invoked from
`make gen-proto` (the buf+docker+gogofaster pipeline is gone). `go.mod`
carries `require github.com/grafana/wiresmith` (runtime `protohelpers`
package) with a local `replace` while the module is unpublished — **remove
the replace and pin a published version before merging**.

## Status per proto

| Proto | Status | Notes |
|---|---|---|
| pkg/tempopb/.wiresmith-proto/{common,resource,trace}/v1 (patched OTel) | DONE | Source of truth is `.wiresmith-proto` (annotated copies of the `pkg/.patched-proto` output; diff and port when the OTel submodule moves). |
| pkg/tempopb/tempo.proto | DONE | stdtime, customtype `PreallocBytes`, customname `Size_`, pointer=true; gRPC stubs via vendored protoc-gen-go-grpc. |
| pkg/tempopb/backendwork.proto | DONE | equal/compare always generated; nullable=false → value field. |
| modules/frontend/v1/frontendv1pb/frontend.proto | DONE | dskit httpgrpc fields carried via customtype envelopes (`httpgrpc_envelope.go`). |
| tempodb/backend/v1/v1.proto | DONE | jsontag/customname/stdtime/customtype map 1:1; embed replaced by named field + hand-written flat MarshalJSON. |

## Test results (2026-06-10)

- `go test -count=1 ./cmd/... ./modules/... ./pkg/... ./tempodb/...`
  (the `make test` package set): all pass except
  - `modules/backendscheduler TestShardedIntegration/sharded_work`: fails
    identically on the unmodified base commit (927ed1b71) — pre-existing,
    not migration fallout.
  - `pkg/usagestats Test_Memberlist`: flaky under parallel package load
    (timing-based memberlist convergence; no proto involvement); passes
    3x consecutively when run alone with `-count=3`.
- Integration (e2e, `make test-e2e`) not run — requires docker images.
- JSON/stored-format compatibility: `tempodb/backend` golden tests
  (`TestBlockMetaJSON*`, fixture-based `TestFixtures`/`TestOriginalFixtures`
  against stored tenant indexes) pass; generated struct json tags were
  diffed against gogo's and match exactly.

## Migration notes

- **Field shapes**: repeated/singular message fields annotated
  `(wiresmith.options.pointer) = true` wherever gogo produced `*T`/`[]*T`,
  to limit call-site churn. gogo `nullable=false` fields map to wiresmith's
  value-type default (annotation deleted). Oneof wrappers hold values
  (`AnyValue_ArrayValue{ArrayValue: v}` not `&v`).
- **gogo reflection interop is gone**: gogo's `proto.Equal/Clone/Merge` and
  `cmp.Diff` cannot traverse wiresmith structs (unexported
  `fieldsPresent [1]uint64`; gogo only skips `XXX_`-prefixed fields).
  Replacements: `test.ProtoEqual`/`test.RequireProtoEqual`
  (pkg/util/test/req.go, wire-bytes equality), `cloneProto` in
  modules/frontend/combiner/common.go (Marshal/Unmarshal round trip;
  normalizes empty top-level slices to nil), and
  `cmpopts.IgnoreUnexported(...)` where cmp.Diff is used.
- **cmp.Diff uses generated Equal on pointers**: `*TimeSeries` etc. now have
  `Equal(any) bool`, which cmp prefers — float comparisons become bit-exact.
  Tests needing tolerance must supply an explicit `cmp.Comparer`
  (tempodb/tempodb_metrics_test.go).
- **No merge-on-unmarshal**: gogo's `Unmarshal` appended repeated fields
  across calls; wiresmith's pre-scan replaces non-empty repeated fields for
  payloads >= 256B (and appends below that). blockbuilder's
  live_traces_iter now unmarshals each batch separately.
- **gRPC**: stubs come from the vendored protoc-gen-go-grpc v1.6
  (requireUnimplemented=true): server impls embed `Unimplemented*Server`
  (frontend.QueryFrontend, livestore.LiveStore, BackendScheduler,
  v1.Frontend, test mocks). Streaming client/server named types are
  aliases of the generic streams, so call sites compile unchanged.
  pkg/gogocodec keeps working: gogo's Marshal/Unmarshal fast paths dispatch
  on the `Marshal() ([]byte, error)` / `Unmarshal([]byte) error` methods,
  which wiresmith generates.
- **jsonpb**: gogo jsonpb still handles Trace/SearchResponse JSON because
  hand-written shims provide `XXX_OneofWrappers` (AnyValue) and register
  enums with the gogo registry (`*_gogo_shim.go`, pkg/tempopb/gogo_shim.go).
- **embed (CompactedBlockMeta)**: solved Tempo-side with a named
  `BlockMeta` field + hand-written flat `MarshalJSON`
  (tempodb/backend/wiresmith_custom.go). Promoted-field fallout:
  ~33 call sites (`x.TenantID` → `x.BlockMeta.TenantID`), mostly in
  tempodb/blocklist, tempodb, cmd/tempo-cli and tests. Manageable; not by
  itself evidence to reopen the wiresmith embed feature, but combined with
  the JSON-flattening footgun it is the most fragile part of the migration.
- **Same-package multi-proto**: tempo.proto + backendwork.proto generate
  into one Go package; wiresmith emits its package-level helpers
  (`maxUnmarshalDepth`, `skipValue`) per file, so `make gen-proto` strips
  the duplicate copy (tools/strip-wiresmith-dup-helpers.py).

## Wiresmith blockers / friction (ranked)

Each entry: where it bites, why wiresmith can't express it, suggested
feature, severity.

1. **Unexported presence bitmap breaks gogo-reflection interop**
   (`proto.Equal/Clone/Merge`, `cmp.Diff`, testify `assert.Equal`).
   - Every mixed gogo/wiresmith graph and every struct-literal vs
     unmarshaled comparison in tests.
   - gogo skips `XXX_`-prefixed fields; the bitmap used to be
     `XXX_fieldsPresent` (interoperated) and is now unexported.
   - Suggested: a compat flag restoring an `XXX_`-prefixed name, or at
     least prominent migration docs. ~40 call sites had to change here.
   - Severity: high during staged migrations; zero once fully migrated.

2. **No multi-proto-per-Go-package support** — duplicate package-level
   helpers (`maxUnmarshalDepth`, `skipValue`) do not compile.
   - pkg/tempopb (tempo.proto + backendwork.proto) — a layout protoc-gen-go
     and gogo support natively.
   - Suggested: emit helpers once per output package (or guard with a
     per-file suffix). Tempo works around it with a post-processing script.
   - Severity: high (compile failure, needs build tooling workaround).

3. **Generated `String()` is `fmt.Sprintf("%v", *m)` — nondeterministic for
   oneofs and not overridable**.
   - `AnyValue.String()` was load-bearing for sort keys / span-set IDs;
     gogo emitted deterministic proto-text. A same-package override is a
     compile error since the generator now always emits String().
   - Suggested: emit a deterministic text form, or an option to suppress
     String() generation.
   - Severity: medium-high — the failure mode is silent nondeterminism, not
     a compile error. Worked around with a hand-written `StableString()`.

4. **Unmarshal replaces non-empty repeated fields when the pre-scan kicks in
   (payload >= 256B) but appends below the threshold**.
   - blockbuilder relied on gogo's merge-on-unmarshal ("unmarshal appends
     batches onto the existing Trace").
   - The size-dependent behavior split is arguably a bug: semantics should
     not depend on payload size. Either always-replace (document loudly) or
     always-append.
   - Severity: medium — silent data loss when relied upon; found only via a
     well-aimed test.

5. **No embed option** (wiresmith-9ks, cancelled) — `CompactedBlockMeta`.
   - Cost measured: ~33 promoted-field call sites + a hand-written flat
     MarshalJSON to preserve stored-JSON compatibility + a `json:"-"` guard
     tag. The JSON-flattening part is the dangerous bit: a tag-only
     marshal would silently produce an incompatible nested shape.
   - Severity: medium for Tempo (one message); fine to keep cancelled
     unless more embed users appear.

6. **customtype on fields referencing externally-generated (gogo) messages
   is the only escape hatch** — generated code calls the wiresmith method
   set on cross-package message types, so a proto importing dskit's
   httpgrpc.proto cannot reference dskit's Go types directly.
   - Worked around with envelope types + Wrap/Unwrap at call sites
     (frontendv1pb). Also: wiresmith leaves an unused Go import for the
     replaced package (goimports in gen step), and the proto build tree
     must contain a gogo-stripped copy of the imported .proto because
     wiresmith cannot parse gogoproto options.
   - Suggested: an option to treat an import as "external gogo package"
     (emit gogo-style Marshal/Size/Unmarshal calls), plus tolerate unknown
     custom options when the defining .proto is resolvable.
   - Severity: medium (one proto in Tempo, contained), but will recur in
     any repo whose protos import dskit/httpgrpc.

7. **No pointer option for map values** — `map<string, ServiceStats>`
   became `map[string]ServiceStats` (gogo: `map[string]*ServiceStats`).
   - Fine for small values; call sites adapted. Would be painful for large
     map values.
   - Severity: low.

8. **No automatic `Name_` mangling when a field collides with a generated
   method** (`uint64 size` vs `Size()`).
   - gogo renamed to `Size_` automatically; wiresmith fails to compile.
     Worked around with `(wiresmith.options.customname) = "Size_"`, which
     also conveniently matches the old gogo API.
   - Severity: low (clear compile error, easy fix), but worth fixing in the
     generator for protoc parity.

9. **Hand-written method collisions with newly generated methods**
   (`TenantIndex.unmarshal`, `QueryRangeRequest.HasInstant`).
   - Not a wiresmith bug per se — consumers must rename. Listed for
     completeness.
   - Severity: low.

## Remaining work

- Run the e2e suite (`make test-e2e`) before merging; cross-version
  compatibility (old querier ↔ new frontend) is exercised there.
- Benchmark the hot paths (distributor PushBytes, query combiners) to
  quantify the wiresmith win on Tempo workloads.
- Replace the local `replace github.com/grafana/wiresmith => ...` with a
  published module version.
- `vendor/modules.txt` pins vtprotobuf one rev newer (pulled transitively
  via the wiresmith module) — sanity-checked, but worth a second look.
- Old gogo annotations are gone from the protos; consider dropping the
  gogoproto vendored package once nothing else imports it (dskit still
  does).
