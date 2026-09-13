---
title: User-configurable overrides
menuTitle: Configure Tempo overrides through the user-configurable overrides API
description: Configure Tempo overrides through the user-configurable overrides API
weight: 300
aliases:
  - ../use-configurable-overrides/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/user-configurable-overrides/
---

# User-configurable overrides

User-configurable overrides in Tempo let you change overrides for your tenant using an API.
Instead of modifying a file or Kubernetes `configmap`, you (and other services relying on Tempo) can use this API to
modify the overrides directly.

## Architecture

User-configurable overrides are stored in an object store bucket managed by Tempo.

![user-configurable-overrides-architecture.svg](../user-configurable-overrides-architecture.svg)

{{< admonition type="note" >}}
We recommend using a different bucket for overrides and traces storage, but they can share a bucket if needed.
When sharing a bucket, make sure any lifecycle rules are scoped correctly to not remove data of user-configurable
overrides module.
{{< /admonition >}}

Overrides of every tenant are stored at `/{tenant name}/overrides.json`:

```
overrides/
├── 1/
│   └── overrides.json
└── 2/
    └── overrides.json
```

Tempo regularly polls this bucket and keeps a copy of the limits in-memory. When requesting the overrides for a tenant,
the overrides module:

1. Checks this override is set in the user-configurable overrides, if so return that value.
2. Checks if this override is set in the runtime configuration (`configmap`), if so return that value.
3. Returns the default value.

### Supported fields

User-configurable overrides are designed to be a subset of the runtime overrides. Refer
to [Overrides](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#overrides) for information about all
overrides.
When you set a field in both the user-configurable overrides and the runtime overrides, the value from the
user-configurable overrides takes priority.

{{< admonition type="note" >}}
In Tempo 2.x, `metrics_generator.processors` user-configurable override field is OR-merged with the runtime overrides list.

In Tempo 3.0, if `processors` is set in user-configurable overrides, that will override the runtime `metrics_generator.processors` config. Setting `processors: []` in user-configurable overrides disables all processors for the tenant.
{{< /admonition >}}

{{< admonition type="warning" >}}
The `local-blocks` processor was removed in Tempo 3.0. TraceQL metrics queries on recent data are now served by the live-store instead. If your overrides reference `local-blocks`, remove it before upgrading.
For details, refer to [Metrics generator](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/).
{{< /admonition >}}

```yaml
[forwarders: <list of strings>]

cost_attribution:
  [dimensions: <map string to string>]

metrics_generator:

  [processors: <list of strings>]
  [collection_interval: <duration>]
  [trace_id_label_name: <string>]
  [ingestion_time_range_slack: <duration>]
  [disable_collection: <bool> | default = false]
  [generate_native_histograms: <classic|native|both> | default = classic]
  [native_histogram_max_bucket_number: <int> | default = 100]
  [native_histogram_bucket_factor: <float> | default = 1.1]
  [native_histogram_min_reset_duration: <duration> | default = 15m]
  [span_name_sanitization: <string>]

  processor:

    service_graphs:
      [histogram_buckets: <list of float>]
      [dimensions: <list of string>]
      [peer_attributes: <list of string>]
      [enable_client_server_prefix: <bool>]
      [enable_messaging_system_latency_histogram: <bool>]
      [enable_virtual_node_label: <bool>]
      [span_multiplier_key: <string>]
      [enable_tracestate_span_multiplier: <bool>]
      [filter_policies: [
        [
          include/include_any/exclude:
            match_type: <string> # options: strict, regex
            attributes:
              - key: <string>
                value: <any>
        ]
      ]]
    span_metrics:
      [histogram_buckets: <list of float>]
      [dimensions: <list of string>]
      [dimension_mappings: <list of mappings {name: <string>, source_labels: <list of string>, join: <string>}>]
      [intrinsic_dimensions: <map string to bool>]
      [filter_policies: [
        [
          include/include_any/exclude:
            match_type: <string> # options: strict, regex
            attributes:
              - key: <string>
                value: <any>
        ]
      ]]
      [enable_target_info: <bool>]
      [target_info_excluded_dimensions: <list of string>]
      [enable_instance_label: <bool>]
      [span_multiplier_key: <string>]
      [enable_tracestate_span_multiplier: <bool>]

    host_info:
      [host_identifiers: <list of string>]
      [metric_name: <string>]

    secret_detection:
      [disabled_rules: <list of native rule IDs to disable> | default = []]
      [custom_rules:
        - id: <string>
          regex: <string>
      ]
```

### Filter policy validation

Filter policies submitted through this API are fully validated. Tempo checks that attribute keys are valid TraceQL identifiers, only `resource` and `span` scopes are supported, regular expression patterns compile, and intrinsic values (`kind`, `status`) use recognized values. Invalid policies are rejected with a descriptive error.

For the complete list of validation rules and supported values, refer to [Filtering](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/#filtering).

### Secret-detection activation

Secret detection is disabled by default and requires both:

- The process-wide `secrets.detection_enabled: true` feature gate.
- `secret-detection` in the tenant's effective `metrics_generator.processors` list.

Opt a tenant in through the existing user-configurable overrides field, retaining any other desired processors:

```json
{
  "metrics_generator": {
    "processors": ["span-metrics", "secret-detection"]
  }
}
```

Runtime overrides can set the same processor list. In Tempo 3.0, an explicit user-configurable list replaces the inherited list. Remove `secret-detection` from the effective list to disable scanning when the processor update is applied; an explicit `processors: []` disables all processors. Deleting the processor override instead restores the inherited list, which might still contain `secret-detection`.

Neither the process-wide gate nor a `metrics_generator.processor.secret_detection` policy alone opts a tenant in. The policy has no per-tenant `enabled` field. Native rule selection, tenant exclusions, and custom rules configure recognition coverage only for tenants whose processor is active.

Once opted in, scanning runs before metrics-generator preprocessing and timestamp filtering, including in sampler deployments. It does not require another metric processor and still runs when `SkipMetricsGeneration` is set.

### Global native rule selection

Deployment operators select the shared native catalog in Tempo configuration, not in tenant overrides. Rule selection does not activate the processor for a tenant:

```yaml
secrets:
  detection_enabled: true
  # Illustrative subset, not a recommended complete catalog.
  enabled_rules:
    - aws-secret-access-key
    - github-pat
    - gemini-authorization-api-key
```

Omitting `enabled_rules`, or setting it to `null`, selects every supported native rule. An explicit list selects exactly those IDs; `enabled_rules: []` selects no native rules while still allowing tenant custom rules. The distinction survives configuration serialization. Unknown or duplicate global IDs are configuration errors even when detection is disabled. Changing this selection requires a process restart.

Global selection controls which native expressions and matcher are compiled and shared. Excluded native implementations remain available in the catalog; they are not deleted. Tenant `disabled_rules` can only subtract from the selected set, never re-enable a globally inactive rule. A known but globally inactive exclusion is a valid no-op; duplicate exclusions still reject the policy. All supported native IDs remain reserved against custom-rule ID collisions.

### Secret-detection policy validation

For an opted-in tenant with the process-wide gate enabled, the metrics-generator validates and compiles the effective `metrics_generator.processor.secret_detection` policy before using it. Persisting a processor-list or policy override does not take effect instantaneously: the overrides delivery cadence, the generator's ten-second processor refresh loop, and any required compilation all apply. Removing `secret-detection` from the effective processor list disables scanning independently of the stored policy.

Each custom rule has exactly two fields: `id` and `regex`. Any match of the regular expression produces a finding for that rule; there are no custom entropy thresholds or capture-specific acceptance checks. Built-in rules use their catalog metadata, entropy thresholds, capture selection, and native validation.

Supported policy bounds are:

- At most 16 custom rules.
- With `secrets.enabled_rules` omitted, all 1,072 supported native rules are selected. `disabled_rules` excludes individual IDs within the global selection; its empty default excludes none. Native exclusions do not consume the 16-custom-rule allowance.
- At most 4096 bytes per regular expression.
- At most 1,048,576 estimated expanded exact-regexp instructions across the custom policy, checked before compiling custom expressions. This bounds compilation work; it is not an exact heap-byte quota.

Malformed regular expressions, duplicate or catalog-conflicting custom IDs, and over-budget policies are rejected as a whole. Unknown, malformed, or duplicate IDs in `disabled_rules` also reject the whole policy; wildcards and custom-rule IDs are not accepted. Errors do not echo submitted IDs or expressions. Existing instances keep their last-known-good policy, including its exclusions and custom rules. An invalid initial policy falls back to the operator-selected native baseline, with no tenant exclusions or custom rules, instead of preventing tenant initialization. A globally empty selection remains empty on fallback. Rejected exclusions are not applied partially. A restarted process has no previous compiled snapshot, so fallback can restore a selected rule that the invalid tenant policy intended to disable, but cannot expand the global selection. Repeated delivery of the same accepted or rejected policy does not repeatedly compile it.

Policies with 17 or more custom rules are rejected, not truncated. Reduce oversized policies in the administrator-owned configuration source. A newly started process with an oversized initial policy uses the selected native baseline without custom rules until a valid policy arrives.

Custom rules are compiled into an immutable shared-candidate plan separate from the native catalog. Every tenant policy for a compiler shares its selected native catalog and matcher; a per-policy bitset skips tenant-excluded candidates before expensive evaluation. Native catalogs and matchers are not cloned for each tenant. Global exclusions avoid native compilation and matcher storage; tenant exclusions do not reclaim shared storage. Native validator expressions initialize once on first use. The same compiler is retained through generator processor replacement. No tenant or policy-revision cache is retained. At most two policy compilations run concurrently per process. Waiting updates re-read the latest effective override, and superseded builds cannot overwrite newer state. Every trace batch uses one complete policy snapshot and a cache bound to that snapshot.

Removing a policy override restores inherited runtime/default policy, which may itself contain exclusions or custom rules; it does not change the processor list. An explicit `secret_detection: {}` replaces the inherited policy with the operator-selected native baseline and no custom rules. The effective policy replaces the inherited policy as a whole; lists do not merge. Within that policy, `disabled_rules: []` excludes no selected native rules and `custom_rules: []` enables no custom rules. Keep all intended exclusions and custom rules when replacing a policy; omitted exclusions restore selected native rules and omitted custom rules lose their coverage. Configure policies through runtime overrides or the existing user-configurable overrides API.

The native catalog has **1,072 supported IDs**. Omitting global rule selection selects all of them for opted-in tenants; it does not enable tenant scanning. For example, `disabled_rules: [generic-api-key]` excludes only that native heuristic if it is globally selected. Every configured custom rule is evaluated while the tenant's processor is active, and disabling a native ID does not release that reserved ID. Native findings appear in lexical rule-ID order, followed by custom findings in configuration order. Global selection-list and tenant exclusion-list order do not affect finding order.

The ambiguity-related rules `generic-api-key`, `heroku-api-key`, `gcp-api-key`, and `openshift-user-token` are selected when global selection is omitted. Broad matching can flag benign cache identifiers, public Heroku OAuth client IDs, intentionally public Google/Firebase keys, OpenShift storage hash names, and other non-credential values with matching shapes. Evaluate representative labeled tenant data; use global selection for fleet-wide coverage/cost decisions and tenant exclusions for tenant-specific noise. **Production zero-false-positive behavior is not established.** Vault rules match modern `hvs.`, `hvb.`, and `hvr.` forms only; single-letter legacy forms are not detected because they collide with ordinary telemetry, not because historic credentials are harmless.

Rules evaluate trace field values only; attribute names are never detector input and are never combined with values. A custom finding follows the rule's expression alone. A native finding follows the built-in rule's expression, capture, entropy, and applicable native validation. These checks do not contact issuers or prove authorization. When enabled, the generic API-key heuristic alone uses its pinned value/match/line candidate filters; those filters do not suppress specific-provider or custom-rule findings. File/commit exclusions and inline ignore directives are not imported.

Candidate-selection and regex-window optimizations are exact accelerators. When equivalence cannot be proven, Tempo evaluates the rule against the complete value. File names alone never establish a finding: source-oriented rules are adapted only when the scanned value contains a sufficient credential-bearing structure, such as a scoped XML section or an explicit private assignment. Unadaptable path-only findings remain excluded.

Policy rejections are reported through value-free logs and `tempo_secret_detection_policy_updates_total{outcome="rejected"}`. Other fixed outcomes are `applied`, `superseded`, and `canceled`. Compilation duration and active work are exposed by `tempo_secret_detection_policy_compilation_duration_seconds` and `tempo_secret_detection_policy_compilations_active`. These process-wide metrics do not use regexes, rule IDs, tenant IDs, hashes, or generations as labels and are not a per-tenant activation-status API. Optimization-cap fallback is not a policy rejection: the exact Go matcher still evaluates the rule.

Operational status/configuration output, logs, traces, and decoding errors omit private policy content. Authorized overrides reads and persistence retain the complete policy; diagnostic redaction does not alter the live configuration.

### Deployment compatibility

Keep deployment selections consistent during rolling restarts: processes can legitimately have different selected sets, and catalog version alone is not an activation fingerprint. Changing or removing a tenant override never expands the owning process's selection.

Persisted JSON permits unknown non-policy fields for forward compatibility, but rejects unsupported `secret_detection` policy and custom-rule members. Custom rules accept only `id` and `regex`; other fields are rejected by strict YAML and JSON policy decoding. Documents with a non-null secret policy must contain exactly one JSON document; trailing whitespace is allowed. Rejections do not expose submitted content. An invalid tenant document retains its last-known-good override or inherits runtime defaults when none exists, without blocking other tenants or startup. A restart cannot recover a previous in-memory policy snapshot.

### Secret-detection coverage

Findings identify matching rules and field kinds, not the native catalog revision or active selection. Use deployed binary and configuration provenance to establish the recognition policy. Tempo supports global native selection, tenant-native exclusions and up to 16 custom rules. Detection executes natively with no production Gitleaks dependency.

#### Supported coverage

The supported catalog recognizes confidential prefixes, explicit private assignments, complete authentication headers and provider-bound requests, connection/configuration credentials, literal credential-bearing commands, and supported private-key structures. Provider-bound validation checks the actual parsed destination and credential role, not nearby brand text. These recognizers target confidential credential carriers, but matching shapes can overlap public identifiers or non-secret values, especially with the four ambiguity-related rules selected by default. A finding is not proof of confidentiality or issuer authenticity. Retired credentials are not treated as nonconfidential merely because their issuer or API is no longer active.

Candidate acceleration preserves the exact accepting regex and structural checks. Single-byte keywords do not disable other rules' position guidance; structural keyword hints cover every regex alternative; mandatory bytes, separator choices and bounded-window runs reject only impossible candidates. Matcher storage remains bounded. A finding reports matching rule IDs rather than a count of unique underlying secrets; overlapping general and provider-specific carriers can produce more than one rule ID for a value.

PostgreSQL detection recognizes nonempty authority or named-query passwords in `postgres://` and `postgresql://` values, including IPv6, multihost, default/socket hosts, and percent-escaped components. Passwordless and empty-password-only URLs are excluded. Structural checks do not establish connectivity or authentication; unrelated unknown query options are not rejected. Complete candidates are limited to 16 KiB, and malformed or oversized candidates are not shortened into apparent matches. This does not introduce blanket decoding or network verification.

The URL rules cover `mysql://`, `mysqlx://`, `mysqlx+srv://`, `mongodb://`, `mongodb+srv://`, `redis://`, `rediss://`, `amqp://`, and `amqps://`. They require a nonempty password in a supported carrier, with component escaping and protocol-specific host forms. Redis also supports its documented `password` query option and follows `redis-cli` when treating lone userinfo as a password; client interpretations can differ. The catalog also supports JDBC drivers, ODBC/ADO connection strings, HTTP/FTP/Snowflake userinfo and other documented carriers. Each rule's fixture evidence states its driver, quoting and size limits; no parser executes commands, resolves DNS, or verifies credentials.

All supported native IDs are reserved, including IDs listed in `disabled_rules`. Custom rules must use different IDs; a collision rejects that custom policy under the last-known-good/default fallback behavior. Finding logs do not carry catalog-version metadata.

A rule match is not proof that a credential was issued, remains valid, or grants access. Each native fixture family records primary sources, established facts, remaining assumptions, and how its synthetic examples were constructed. Prefix/example-backed candidates must not be described as complete provider-format validation. GitHub/npm checksum details and the complete Slack user-token alphabet remain explicitly unresolved rather than guessed.

`CatalogRuleIDs()` in `pkg/secrets` lists all supported native IDs. This is the inventory, not a process's explicit selection or a tenant's post-exclusion active set. Context-dependent families require the relevant carrier inside the scanned value. If selected, generic detection can flag unknown-provider candidates with credential-like assignment context and the configured entropy/filter conditions; it does not identify arbitrary bare application secrets. No match proves issuance or authenticity.

Removing trailing Base64 padding from a legacy Grafana key does not remove credential information. Tempo accepts strict padded and unpadded standard Base64 when the complete typed JSON credential remains, while retaining field-width and decoding checks. The generic heuristic is selected by default despite its known ambiguity, including collisions with benign cache identifiers.

Mixed Unicode and invalid-byte values can use keyword-guided execution when a rule is proven to begin with an exact ASCII keyword prefix. Evaluation uses UTF-8-safe byte bounds and preserves the original preceding rune for assertions and captures. Rules without this proof use the full-value fallback.

Custom rules with identical regex expressions may share immutable compiled internals within a policy. Rule IDs and findings remain independent, and validation/resource limits still account for every configured rule before reuse. These custom plans are not cached globally across tenants or historical policy revisions.

## API

All API requests are handled on the `/api/overrides` endpoint. The module supports `GET`, `POST`, `PATCH`, and `DELETE`
requests. If you set [`http_api_prefix`](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#server) in your Tempo configuration, prepend it to the path (for example, `/tempo/api/overrides`).

This endpoint is tenant-specific. If Tempo is run in multitenant mode, all requests should have an appropriate
`X-Scope-OrgID` header.

If the tenant is run in distributed mode, only the query-frontend will accept API requests.

### Operations

#### GET /api/overrides

Returns the current overrides and its version.

Query-parameters:

- `scope`: whether to return overrides from the API only `api` or merge it with the runtime overrides `merged`. Defaults
  to `api`.

Example:

```shell
curl -X GET -v -H "X-Scope-OrgID: 3" http://localhost:3100/api/overrides\?scope=merged
```

#### POST /api/overrides

Update the overrides with the given payload. Note this overwrites any existing overrides.

Example:

```shell
curl -X POST -v -H "X-Scope-OrgID: 3" -H "If-Match: 1697726795401423" http://localhost:3100/api/overrides --data "{}"
```

#### PATCH /api/overrides

Update the existing overrides by patching it with the payload.
It follows the JSON merge patch protocol ([RFC 7386](https://datatracker.ietf.org/doc/html/rfc7386)).

Example:

```shell
curl -X PATCH -v -H "X-Scope-OrgID: 3" http://localhost:3100/api/overrides --data "{\"forwarders\":null}"
```

#### DELETE /api/overrides

Delete the existing overrides.

Example:

```shell
curl -X DELETE -H "X-Scope-OrgID: 3" -H "If-Match: 1697726795401423" http://localhost:3100/api/overrides
```

### Versioning

To handle concurrent read and write operations, the backend stores the overrides with a version.
Whenever the overrides are returned, the response has an Etag header with the current version.

```shell
$ curl -v http://localhost:3100/api/overrides
...
< HTTP/1.1 200 OK
< Content-Type: application/json
< Etag: 1697726795401423
< Date: Wed, 07 Feb 2024 17:49:04 GMT
< Content-Length: 118
...
```

Requests that modify or delete overrides need to pass the current version using the `If-Match` header:

```shell
curl -X POST -H "If-Match: 1697726795401423" http://localhost:3100/api/overrides --data "..."
```

This example uses overrides in the `overrides.json` file with the location in `pwd`:

```shell
curl -X POST -H "X-Scope-OrgID: 3" -H "If-Match: 1697726795401423" http://localhost:3100/api/overrides --data @overrides.json
```

If the version doesn't match the version in the backend, the request is rejected with HTTP error 412.

### Conflicting runtime overrides check

Overrides set through the user-configurable overrides take priority over runtime overrides.
This can lead to misleading scenarios because a value set in the runtime overrides is not actively being used.

To warn users about preexisting runtime overrides, there is an optional check for conflicting runtime overrides.
If enabled requests are rejected if:

1. There are no user-configurable overrides yet for this tenant.
2. There are runtime overrides set that contain overrides present in the user-configurable overrides.

The check can be enabled in the configuration:

```yaml
overrides:
  user_configurable_overrides:
    api:
      check_for_conflicting_runtime_overrides: true
```

You can bypass this check by setting the query parameter `skip-conflicting-overrides-check=true`:

```shell
curl -X POST -H "If-Match: 1697726795401423" http://localhost:3100/api/overrides?skip-conflicting-overrides-check=true --data "..."
```
