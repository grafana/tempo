# Tempo → wiresmith protobuf compiler migration

Tracking document for replacing protoc-gen-gogofaster with
[wiresmith](https://github.com/grafana/wiresmith) for Tempo's own protos.
Wire format is unchanged (standard protobuf, binary-compatible with gogo);
only the generated Go API shape changes.

Toolchain: `wiresmith` CLI built from the public `github.com/grafana/wiresmith`
`databases` branch (`v0.0.0-20260612130815-854b4c6268c2`, commit 854b4c6 —
the `UnmarshalNoPrescan` compiler, see the §`UnmarshalNoPrescan` adoption
below), invoked from `make gen-proto` (the buf+docker+gogofaster pipeline is
gone). The repo is public and go-installable: `go.mod` carries `require
github.com/grafana/wiresmith` (runtime `protohelpers` package) pinned to that
published `databases` pseudo-version — no `replace` and no private-module env
needed. `make gen-proto` reproduces the committed output byte-identically
(regen-vs-pinned now consistent: install the pinned binary with
`go install github.com/grafana/wiresmith/cmd/wiresmith@v0.0.0-20260612130815-854b4c6268c2`
and the generated `.pb.go` are byte-for-byte unchanged).

Remaining toolchain step: once the `databases` branch merges into wiresmith
`main`, bump go.mod to the resulting `main` pseudo-version. The merge will be a
squash, which orphans commit 854b4c6, but the Go module proxy keeps the orphaned
pseudo-version fetchable, so the pin stays installable until the bump lands.

Note: `(wiresmith.options.enum_no_prefix)` (gogo goproto_enum_prefix=false
parity) exists since @698587a but is not needed — Tempo's gogo protos all
used prefixed enum constants.

## Status per proto

| Proto | Status | Notes |
|---|---|---|
| pkg/tempopb/.wiresmith-proto/{common,resource,trace}/v1 (patched OTel) | DONE | Source of truth is `.wiresmith-proto` (annotated copies of the `pkg/.patched-proto` output; diff and port when the OTel submodule moves). |
| pkg/tempopb/tempo.proto | DONE | stdtime, customtype `PreallocBytes`, customname `Size_`, pointer=true; gRPC stubs via wiresmith's built-in grpc generator (emits a `protoc-gen-go-grpc v1.6.0`-labelled header for compat; no external tool vendored in this repo). |
| pkg/tempopb/backendwork.proto | DONE | equal/compare always generated; nullable=false → value field. |
| modules/frontend/v1/frontendv1pb/frontend.proto | DONE | dskit httpgrpc fields carried via customtype envelopes (`httpgrpc_envelope.go`). |
| tempodb/backend/v1/v1.proto | DONE | jsontag/customname/stdtime/customtype map 1:1; embed replaced by named field + hand-written flat MarshalJSON. |

## Public-toolchain brush-up (2026-06-12)

Re-pinned go.mod from the c489c2b pseudo-version to the public
`v0.0.0-20260611164808-4f41063d76a2` (4f41063) and re-ran `make gen-proto`.
Two API-neutral commits land between c489c2b and 4f41063 (#136 CI, #137
generator refactor), so regen was expected byte-identical — and was for
every `pkg/tempopb/*` file and the vendored `protohelpers` (all hashes
unchanged). **Exception:** `modules/frontend/v1/frontendv1pb/frontend.pb.go`
and `tempodb/backend/v1.pb.go` changed — they still carried the *old*
exact-fit pre-scan grow (`grown := make(..., need); copy(...)`). The earlier
pre-scan-fix regen (commit 0b331f994) only re-ran the `pkg/tempopb` step, so
these two files missed the `wiresmith-zlce` O(n²) fix. The full regen on
4f41063 applies the amortized-append form (`if len(...)==0 && cap(...)<c`)
everywhere; no old form remains. Beneficial, kept.

Workaround review against now-shipped compiler features (#117 customtype-on-
message, #118 stdduration, #119 casttype, #121 Has<F>(), #133 transitive `-M`,
#104 customname): **nothing removed.** Each remaining workaround is either an
in-use shipped feature (stdtime, customname `Size_`, customtype envelopes,
generated `HasInstant`) or tied to a deliberate decision / open gap:
- `httpgrpc_envelope.go` + the frontend `customtype` fields — required bridge
  for dskit's *gogo-generated* httpgrpc types (wiresmith calls its own
  `*Wiresmith` method set, which gogo types lack). #117 is what lets the
  field *declare* the customtype; the envelope is still the only escape hatch
  (blocker #6, open). Kept.
- The frontend `-M "…/httpgrpc.proto=…/httpgrpc"` transitive-import pin — #133
  makes it *honored*, but because the customtype envelopes replace the
  httpgrpc field types the generated package never imports dskit, so this `-M`
  is now provably a no-op for output (verified: dropping it yields a
  byte-identical `frontend.pb.go`). Kept as harmless intent-documentation and
  a guard against a future non-customtype field; the staged gogo-stripped
  httpgrpc copy is still load-bearing (wiresmith must resolve the import).
- `Size_` customname (blocker #8), embed-replacement `wiresmith_custom.go`
  (blocker #5), `no_presence_all`, the `*_gogo_shim.go` jsonpb interop +
  `StableString` (blocker #3) — all deliberate, all kept.

## Test results (2026-06-12, public toolchain 4f41063)

- `go test -count=1 ./cmd/... ./modules/... ./pkg/... ./tempodb/...`
  (the `make test` package set, 86 packages): all pass. The
  `modules/backendscheduler TestShardedIntegration/sharded_work` case,
  which fails identically on the unmodified base commit (927ed1b71) and is
  pre-existing (not migration fallout), did not trip this run — it is
  load/timing-dependent and passed here. (`pkg/usagestats Test_Memberlist`
  has flaked under parallel load on earlier runs; passed here.)
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
- ~~Benchmark the hot paths~~ — done for the ingest decode path, see
  **Benchmarks** below.
- ~~Replace the local `replace github.com/grafana/wiresmith => ...` with a
  published module version.~~ Done — pinned to the public `databases`
  pseudo-version `v0.0.0-20260612130815-854b4c6268c2` (854b4c6, the
  `UnmarshalNoPrescan` compiler); no `replace` remains. Regen-vs-pinned is
  consistent (committed `.pb.go` byte-identical to a fresh regen with that
  binary). Remaining: bump to the `main` pseudo-version once `databases` merges
  to wiresmith `main` (squash orphans 854b4c6; the module proxy keeps it
  fetchable in the meantime).
- `vendor/modules.txt` pins vtprotobuf one rev newer (pulled transitively
  via the wiresmith module) — sanity-checked, but worth a second look.
- Old gogo annotations are gone from the protos; consider dropping the
  gogoproto vendored package once nothing else imports it (dskit still
  does).

## Benchmarks (Apple M4 Pro, benchstat-grade — DB-9, 2026-06-11)

`pkg/ingest` decode benchmarks, gogo baseline (7c61b2b70b) vs wiresmith
`wiresmith` branch. Method: two `go test -c` binaries **alternated** 20 rounds
so thermal drift cancels; `benchstat`, all deltas p=0.000 except OTLP time.

| Bench | gogo sec/op | wiresmith sec/op | Δ time | B/op Δ | allocs Δ |
|---|---|---|---|---|---|
| GeneratorDecoderOTLP       | 92.46µ | 90.53µ | ~ (p=0.20) | −6.88% | −5.34% |
| GeneratorDecoderPushBytes  | 172.6µ | 183.3µ | +6.14% | −1.02% | +2.67% |
| EncodeDecode               | 165.5µ | **3800µ** | **+2197%** | **+6788%** (1.25Mi→86Mi) | +0.10% |

- **OTLP decode**: parity (slightly fewer allocs/bytes). The decoder resets
  `trace.ResourceSpans[:0]` before each `Unmarshal`.
- **PushBytes decode**: +6.1% wall, ~flat memory. `PushBytesDecoder` calls
  `Reset()` (len→0, cap retained), so the pre-scan finds enough capacity and
  does no realloc — the +6% is purely the pre-scan's extra linear *scan* over
  the payload (same class as mimir RW2, bead `wiresmith-bobw` / DB-18).
- **EncodeDecode**: catastrophic — +2197% time, 86 MiB/op (67×). This bench
  reuses one bare `ingest.Decoder` and calls `Decode` (→ `PushBytesRequest.
  Unmarshal`) repeatedly **without** `Reset()`. Each decode appends, so
  `len(m.Traces)` grows every call; the pre-scan grows the slice with
  **exact-fit** capacity (`make([]T, len, len+c)`), reallocating+copying the
  entire growing backing array every call ⇒ **O(n²)**. gogo's plain `append`
  doubles capacity (amortized O(n)) so the identical no-Reset usage stays
  bounded. Proven by an isolation build (pre-scan forced off): wiresmith
  collapses to **161µs / 1.33 MiB — a dead match with gogo** (165µs / 1.30 MiB).

The EncodeDecode blowup was a genuine perf cliff (not just bench misuse:
`ingest.Decoder.Decode` is public and doesn't reset). Filed as
**`wiresmith-zlce`** (P1) and **FIXED** (Option A): `emitPreScan` now reserves
the slice only when `len(m.X)==0`; merges into a populated slice fall back to
amortized append (gogo-equivalent). The `pkg/tempopb/*.pb.go` here are
regenerated with that fixed compiler (wiresmith-databases @39ef729 →
`grafana/wiresmith:pre-scan-tempo-fix`). Re-bench (gogo 7c61b2b70b vs fixed):

| Bench | gogo | wiresmith (fixed) | vs gogo |
|---|---|---|---|
| EncodeDecode | 173.7µ / 1.235Mi | 178.1µ / 1.247Mi | **parity** (p=0.10 / 0.14) |
| GeneratorDecoderOTLP | 91.7µ | 87.6µ | −4.4% |
| GeneratorDecoderPushBytes | 175.3µ | 182.0µ | +3.8% |

EncodeDecode dropped from 3.8ms / 86Mi to gogo parity (~21× faster, ~69× less
memory); the O(n²) is gone. PushBytes +3.8% is the *separate* scan-cost class
(`wiresmith-bobw` / DB-18), untouched by this fix. Distinct also from the
prealloc-reuse *security* analysis (`wiresmith-u4qg` / DB-6).

### Re-bench on the public toolchain (2026-06-12, 4f41063)

Re-ran the same alternated `go test -c` method (20 rounds, idle M4 Pro) after
re-pinning to the public `v0.0.0-20260611164808-4f41063d76a2` and full
`make gen-proto` (gogo baseline 7c61b2b70b):

| Bench | gogo sec/op | wiresmith sec/op | Δ time | B/op Δ | allocs Δ |
|---|---|---|---|---|---|
| EncodeDecode              | 182.2µ / 1.250Mi | 192.0µ / 1.232Mi | +5.40% (p=0.000) | ~ (p=0.34) | ~ (2004, equal) |
| GeneratorDecoderOTLP      | 99.45µ | 95.10µ | −4.37% (p=0.04) | −6.24% | −4.97% |
| GeneratorDecoderPushBytes | 192.4µ | 201.4µ | +4.66% (p=0.000) | −1.05% | +2.51% |

- **EncodeDecode**: memory and allocs are at gogo parity (1.23–1.25 MiB, 2004
  allocs) — the `wiresmith-zlce` O(n²) fix is confirmed holding on the public
  compiler (no 86 MiB blowup). The +5.4% steady-state *time* is a small, stable
  delta (vs the earlier ~+2.4% reading; run-to-run, not the perf cliff). The
  load-bearing result — bounded O(n) memory — is intact.
- **PushBytes**: +4.66% time / +2.51% allocs is the same DB-18
  (`wiresmith-bobw`) pre-scan linear-scan cost noted above (was +3.8%); not a
  new regression.
- **OTLP**: faster than gogo on all three axes.

### `UnmarshalNoPrescan` adoption at reuse sites (2026-06-12)

The PushBytes (+4.66%) and EncodeDecode (+5.40% time, fluctuating up to the
+7.9% seen in a later run) regressions above were both 100% the generated
*top-level* pre-scan in `(*PushBytesRequest).unmarshal`. The prealloc the
pre-scan feeds never fires on these paths: the `ingest.Decoder` REUSES its
`PushBytesRequest` (`Reset()` does `Traces[:0]/Ids[:0]`, retaining cap; the
EncodeDecode bench doesn't even `Reset()`), so the scan is pure overhead. OTLP
(the −2.9%/−4.4% control) instead gains from the *nested* pre-scans inside
`Trace`/`ResourceSpans` and must keep them.

The wiresmith `databases` compiler (@854b4c6) now emits
`UnmarshalNoPrescan(dAtA []byte) error` on every pre-scan-bearing message. It
calls `unmarshal(dAtA, -1)`: the generated top-level guard is now
`if l >= 256 && depth >= 0`, so depth `−1` skips ONLY the top-level pre-scan;
nested messages recurse via `UnmarshalWithDepth(b, depth+1)` (which clamps a
negative depth to 0), so the first nested level lifts to depth 0 and pre-scans
normally. Default `Unmarshal` (depth 0) is byte-for-byte unchanged.

Adopted at exactly one reuse site — `pkg/ingest/encoding.go`'s
`(*Decoder).Decode`, `d.req.Unmarshal(data)` → `d.req.UnmarshalNoPrescan(data)`
— which both regressing benchmarks share (the EncodeDecode bench reuses the
bare `Decoder`; `PushBytesDecoder` wraps the same `Decoder`). `PushBytesRequest`
has no nested message fields, so no nested pre-scans are lost. The OTLP path
(`OTLPDecoder`, `(*Trace).Unmarshal`) is left on plain `Unmarshal` to keep its
nested pre-scans and its win.

Re-bench, same alternated 20-round `go test -c` method (idle M4 Pro, gogo
baseline 7c61b2b70b) with the `databases`@854b4c6 binary regen + the call-site
change:

| Bench | gogo sec/op | wiresmith sec/op | Δ time | B/op Δ | allocs Δ |
|---|---|---|---|---|---|
| EncodeDecode              | 148.2µ / 1.227Mi | 150.0µ / 1.228Mi | ~ (p=0.640) | ~ (p=0.659) | ~ (2004, equal) |
| GeneratorDecoderOTLP      | 89.74µ | 88.30µ | ~ (p=0.068) | −6.75% | −5.57% |
| GeneratorDecoderPushBytes | 168.6µ | 166.6µ | −1.22% (p=0.008) | −1.09% | +2.67% |

- **EncodeDecode**: time is now at gogo parity (was +5.40%/+7.9%); memory holds
  at gogo parity (1.227→1.228 MiB, 2004 allocs exactly equal) — the
  `wiresmith-zlce` O(n²) fix is still intact, the pre-scan removal didn't
  disturb it.
- **PushBytes**: −1.22% time — now marginally *faster* than gogo (was +4.66%);
  the DB-18 (`wiresmith-bobw`) linear-scan cost is removed on this path. B/op
  −1.09%; allocs +2.67% (unchanged from before — not pre-scan related).
- **OTLP**: control, unchanged call site; still faster than gogo on bytes
  (−6.75%) and allocs (−5.57%), time at parity. The nested pre-scan win is
  retained.

Both remaining time regressions are eliminated at parity (n=20) with the OTLP
win kept and EncodeDecode memory/allocs at gogo parity. Other reuse-decode
sites were deliberately NOT changed: `blockbuilder/live_traces_iter.go`'s
merge-by-unmarshal (kept byte-identical to upstream) and
`livestore/instance.go`'s `[:0]`-reset `Trace` decode are candidates left for a
separate decision.
