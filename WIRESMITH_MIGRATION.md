# Tempo → wiresmith protobuf compiler migration

Tracking document for replacing protoc-gen-gogofaster with
[wiresmith](https://github.com/grafana/wiresmith) for Tempo's own protos.
Wire format is unchanged (standard protobuf, binary-compatible with gogo);
only the generated Go API shape changes.

Toolchain: `wiresmith` CLI built from the `databases` branch (@75caef9),
invoked from `make gen-proto` (the buf+docker+gogofaster pipeline is gone).
`go.mod` carries `require github.com/grafana/wiresmith` (runtime
`protohelpers` package) with a local `replace` to the wiresmith-databases
checkout while the branch is unpublished — **remove the replace and pin a
published version before merging**. `make gen-proto` reproduces the
committed output byte-identically.

Note: `(wiresmith.options.enum_no_prefix)` (gogo goproto_enum_prefix=false
parity) exists since @698587a but is not needed — Tempo's gogo protos all
used prefixed enum constants.

## Status per proto

| Proto | Status | Notes |
|---|---|---|
| pkg/tempopb/.wiresmith-proto/{common,resource,trace}/v1 (patched OTel) | DONE | Source of truth is `.wiresmith-proto` (annotated copies of the `pkg/.patched-proto` output; diff and port when the OTel submodule moves). |
| pkg/tempopb/tempo.proto | DONE | stdtime, customtype `PreallocBytes`, customname `Size_`, pointer=true; gRPC stubs via vendored protoc-gen-go-grpc. |
| pkg/tempopb/backendwork.proto | DONE | equal/compare always generated; nullable=false → value field. |
| modules/frontend/v1/frontendv1pb/frontend.proto | DONE | dskit httpgrpc fields carried via customtype envelopes (`httpgrpc_envelope.go`). |
| tempodb/backend/v1/v1.proto | DONE | jsontag/customname/stdtime/customtype map 1:1; embed replaced by named field + hand-written flat MarshalJSON. |

## Test results (2026-06-10, databases-branch toolchain)

- `go test -count=1 ./cmd/... ./modules/... ./pkg/... ./tempodb/...`
  (the `make test` package set, 85 packages): all pass except
  `modules/backendscheduler TestShardedIntegration/sharded_work`, which
  fails identically on the unmodified base commit (927ed1b71) —
  pre-existing, not migration fallout. (`pkg/usagestats Test_Memberlist`
  flaked once on the previous run under parallel load; passes here.)
- e2e (docker): the full `make test-e2e` suite set — `integration/api`,
  `operations`, `limits`, `metrics-generator`, `storage`, `util` — passes
  with locally built grafana/tempo + tempo-query images. The api suite
  exercises the customtype-enveloped httpgrpc bidi stream, streaming gRPC
  search, and the JSON+protobuf HTTP APIs, and caught one real bug
  (pkg/httpclient used golang/protobuf jsonpb, see Migration notes) plus
  one nil-vs-empty decode difference in a test helper.
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
- **gogo reflection interop**: restored on the databases branch — the
  presence bitmap is `XXX_fieldsPresent`, which gogo's reflection-based
  `proto.Equal/Clone/Merge` skip. The ~25 call sites that had been switched
  to wire-bytes comparison are back on gogo `proto.Equal` (upstream form).
  Note the field is exported: testify `require.Equal` and `cmp.Diff` DO see
  it, so struct-literal vs unmarshaled comparisons still need
  `test.RequireProtoEqual` (wire-bytes), the generated `Equal()`, or
  `cmpopts.IgnoreFields(T{}, "XXX_fieldsPresent")` / a `cmp.FilterPath`
  (tempodb/backend/cmp_test.go). Kept on purpose (better, not workarounds):
  `cloneProto` in modules/frontend/combiner/common.go (generated
  marshal/unmarshal round trip instead of reflection Clone; normalizes
  empty top-level slices to nil) and vulture's wire-bytes `equalTraces`.
- **cmp.Diff uses generated Equal on pointers**: `*TimeSeries` etc. now have
  `Equal(any) bool`, which cmp prefers — float comparisons become bit-exact.
  Tests needing tolerance must supply an explicit `cmp.Comparer`
  (tempodb/tempodb_metrics_test.go).
- **Merge-on-unmarshal**: fixed upstream (@c471fd6, uniform append/merge
  semantics on both sides of the pre-scan threshold). blockbuilder's
  live_traces_iter is reverted to upstream's merge-by-unmarshal — the
  file is byte-identical to base main again.
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
  The **golang/protobuf** jsonpb shim is NOT usable: it routes through
  protoreflect field-level reflection, which wiresmith messages reject
  with a panic. pkg/httpclient and modules/querier were switched to gogo
  jsonpb (found by the e2e api suite — a unit-test blind spot). One
  behavior nit: gogo jsonpb decodes an empty JSON array into an empty
  non-nil slice where the golang decoder left nil (integration helper
  adjusted).
- **no_presence_all**: adopted on every Tempo proto. The DB-10 exported
  bitmap is untagged, so encoding/json serialized
  `"XXX_fieldsPresent":[0]` into stored meta.json/tenant indexes, the
  `dc=` query param, and work-shard files; dropping the bitmap fixes that
  at the root and restores gogo's declared-fields-only layout (Tempo
  never had Has*/present-but-empty semantics under gogo).
- **embed (CompactedBlockMeta)**: solved Tempo-side with a named
  `BlockMeta` field + hand-written flat `MarshalJSON`
  (tempodb/backend/wiresmith_custom.go). Promoted-field fallout:
  ~33 call sites (`x.TenantID` → `x.BlockMeta.TenantID`), mostly in
  tempodb/blocklist, tempodb, cmd/tempo-cli and tests. Manageable; not by
  itself evidence to reopen the wiresmith embed feature, but combined with
  the JSON-flattening footgun it is the most fragile part of the migration.
- **Same-package multi-proto**: tempo.proto + backendwork.proto generate
  into one Go package; the databases branch moved the unmarshal helpers
  into `protohelpers` (`SkipValue`/`MaxUnmarshalDepth`), so this compiles
  natively — the former strip script is deleted.

## Wiresmith blockers / friction (ranked)

Each entry: where it bites, why wiresmith can't express it, suggested
feature, severity.

1. **FIXED on the databases branch (@9407011)**: presence bitmap renamed
   `XXX_fieldsPresent` — gogo `proto.Equal/Clone/Merge` interop restored;
   the ~25 wire-bytes-comparison call sites were reverted to upstream's
   gogo `proto.Equal`. Follow-up found during adoption: the exported
   bitmap carries **no struct tag**, so `encoding/json` serializes
   `"XXX_fieldsPresent":[0]` into every JSON surface (gogo tags its XXX_
   fields `json:"-"`). Tempo sidestepped it by adopting
   `(wiresmith.options.no_presence_all)` on every proto (gogo parity —
   Tempo never had Has*/present-but-empty semantics), which also let the
   remaining testify/cmp bitmap workarounds revert to upstream forms.
   The bitmap carries `json:"-"` since @75caef9, so even protos that keep
   it are JSON-safe now.

2. **FIXED on the databases branch (@5f1416f)**: `skipValue`/
   `maxUnmarshalDepth` live in `protohelpers` and are referenced
   qualified; multiple .proto files per Go package compile natively.
   tools/strip-wiresmith-dup-helpers.py and its gen-proto step are
   deleted.

3. **Generated `String()` is `fmt.Sprintf("%v", *m)` — nondeterministic for
   oneofs and not overridable**.
   - `AnyValue.String()` was load-bearing for sort keys / span-set IDs;
     gogo emitted deterministic proto-text. A same-package override is a
     compile error since the generator now always emits String().
   - Suggested: emit a deterministic text form, or an option to suppress
     String() generation.
   - Severity: medium-high — the failure mode is silent nondeterminism, not
     a compile error. Worked around with a hand-written `StableString()`.

4. **FIXED on the databases branch (@c471fd6, DB-13)**: unmarshal into a
   non-empty message appends repeated elements / merges maps uniformly on
   both sides of the 256B pre-scan threshold. blockbuilder's adaptation
   is reverted; gogo merge-by-unmarshal semantics hold again.

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
     (frontendv1pb). The unused-import leak for customtype-replaced
     packages is fixed on the databases branch (@d143d4f; the goimports
     gen step is gone). Still required: the proto build tree must contain
     a gogo-stripped copy of the imported .proto because wiresmith cannot
     parse gogoproto options.
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

- e2e: all six suites pass with locally built images. Still worth a
  mixed-version cluster test (old querier ↔ new frontend) before
  merging — the suites here run a single version.
- Benchmark the hot paths (distributor PushBytes, query combiners) to
  quantify the wiresmith win on Tempo workloads.
- Replace the local `replace github.com/grafana/wiresmith => ...` with a
  published module version.
- `vendor/modules.txt` pins vtprotobuf one rev newer (pulled transitively
  via the wiresmith module) — sanity-checked, but worth a second look.
- Old gogo annotations are gone from the protos; consider dropping the
  gogoproto vendored package once nothing else imports it (dskit still
  does).
