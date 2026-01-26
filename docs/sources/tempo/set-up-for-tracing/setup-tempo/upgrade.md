---
title: Upgrade your Tempo installation
menuTitle: Upgrade
description: Upgrade your Grafana Tempo installation to the latest version.
weight: 510
aliases:
  - ../../setup/upgrade/ # /docs/tempo/<TEMPO_VERSION>/setup/upgrade
---

# Upgrade your Tempo installation

<!-- vale Grafana.We = NO -->
<!-- vale Grafana.Will = NO -->
<!-- vale Grafana.Timeless = NO -->

You can upgrade a Tempo installation to the next version.
However, any release has the potential to have breaking changes.
We recommend testing in a non-production environment prior to rolling these changes to production.

The upgrade process changes for each version, depending upon the changes made for the subsequent release.

This upgrade guide applies to on-premise installations and not for Grafana Cloud.

For detailed information about any release, refer to the [Release notes](https://grafana.com/docs/tempo/<TEMPO_VERSION>/release-notes/).

{{< admonition type="tip" >}}
You can check your configuration options using the [`status` API endpoint](https://grafana.com/docs/tempo/<TEMPO_VERSION>/api_docs/#status) in your Tempo installation.
{{< /admonition >}}

## Upgrade to Tempo 2.10

When upgrading to Tempo 2.10, be aware of these considerations and breaking changes.

When [upgrading](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/) to Tempo 2.10, be aware of these considerations and breaking changes.

### Busybox removed from Tempo image

The Tempo container image no longer includes busybox. This change reduces the image size and attack surface, preventing future busybox-related vulnerabilities from affecting Tempo deployments. [[PR 5717](https://github.com/grafana/tempo/pull/5717)]

The image switched from `gcr.io/distroless/static-debian12:debug`, which includes busybox, to `gcr.io/distroless/static-debian12`, which doesn't. The busybox shell and utilities are no longer available inside the running container.

You can no longer `exec` into the Tempo container with a shell. Commands like `kubectl exec -it <pod> -- sh` or `docker exec -it <container> sh` will fail.

To debug a running Tempo container, use one of these alternatives:
- Kubernetes ephemeral debug containers (`kubectl debug`)
- Docker Desktop or other container runtime tools that support shell injection for distroless images

If you have custom Docker Compose files or scripts that use the Tempo image for shell operations (such as running `chown` in an init container), update them to use a separate `busybox:latest` image for those tasks.

Tempo's runtime behavior, configuration options, and APIs are unchanged.

### vParquet format changes

This release includes breaking changes to vParquet block format support and deprecates older formats ahead of Tempo 3.0.

In preparation for Tempo 3.0, make sure you're using vParquet4 or higher.

#### Enable vParquet5 (optional)

vParquet5 is now production-ready and available as an optional upgrade. vParquet4 remains the default block format. [[PR 6219](https://github.com/grafana/tempo/pull/6219)]

To enable vParquet5, set the block version in your [storage configuration](/docs/tempo/<TEMPO_VERSION>/configuration/parquet/):

```yaml
storage:
  trace:
    block:
      version: vParquet5
```

Once enabled, Tempo writes new blocks in vParquet5 format while continuing to read existing vParquet4 blocks. No migration of existing data is required.

#### vParquet2 removed

vParquet2 encoding has been completely removed from Tempo 2.10. Tempo can no longer read vParquet2 blocks. [[PR 6071](https://github.com/grafana/tempo/pull/6071)]

No action is required if you've used default settings. The default block format migrated away from vParquet2 several releases ago.

Action is required if your storage configuration explicitly specifies vParquet2. Before upgrading, verify your configuration doesn't specify vParquet2. If it does:
  - Update to a previous Tempo release (2.9 or earlier) configured for vParquet3 or higher
  - Wait for all existing vParquet2 blocks to expire and be deleted from backend storage
  - Then upgrade to Tempo 2.10

Upgrading to 2.10 while vParquet2 blocks still exist in storage will cause read errors.

#### vParquet3 and vParquet2 schemas deprecated

TempoDB schemas vParquet3 and vParquet2 are now deprecated and will be removed in Tempo 3.0. [[PR 6198](https://github.com/grafana/tempo/pull/6198)]

If you're using vParquet3, plan your migration to vParquet4 or higher before upgrading to Tempo 3.0.

### Breaking changes

Be aware of these breaking changes when upgrading to Tempo 2.10:

- Tenant ID validation: Tempo now validates tenant IDs in the frontend and distributor components, using the same validation rules as Grafana Mimir. Previously, Tempo accepted any tenant ID value. Valid tenant IDs may only contain alphanumeric characters (a-z, A-Z, 0-9) and special characters (`!`, `-`, `_`, `.`, `*`, `'`, `(`, `)`), with a maximum length of 150 characters. Empty tenant IDs and the values `.` or `..` are rejected. Requests with invalid tenant IDs return an error. [[PR 5786](https://github.com/grafana/tempo/pull/5786)]
- SearchTagsV2WithRange API change: The Tempo HTTP client `SearchTagsV2WithRange` function signature has changed. The function previously accepted only `start` and `end` parameters. It now requires two additional parameters: `scope` and `query`. This change affects Go code that imports the `pkg/httpclient` package. To maintain existing behavior, pass empty strings for the new parameters: `SearchTagsV2WithRange("", "", start, end)`. [[PR 6088](https://github.com/grafana/tempo/pull/6088)]
- Go version upgrade: Tempo 2.10 upgrades to Go 1.25.5, which may affect custom builds or deployments with specific Go version requirements. (PRs [#5939](https://github.com/grafana/tempo/pull/5939) [#6001](https://github.com/grafana/tempo/pull/6001), [#6096](https://github.com/grafana/tempo/pull/6096), [#6089](https://github.com/grafana/tempo/pull/6089))

## Upgrade to Tempo 2.9

When upgrading to Tempo 2.9, be aware of these considerations and breaking changes.

### RPM and DEB packages discontinued

Tempo no longer publishes RPM and DEB packages due to an internal change to the handling of signing keys. This can be restored if customers need these packages. ([PR 5684](https://github.com/grafana/tempo/pull/5684))

### Migrated Vulture and integration tests to OTLP exporter

In this release, we've migrated Tempo Vulture and Integration Tests from the deprecated Jaeger agent/exporter to the standard OTLP exporter. Vulture now pushes traces to the Tempo OTLP GRCP endpoint. ([PR 5058](https://github.com/grafana/tempo/pull/5058))

### Bucket calculation changes

TraceQL metrics buckets are now calculated based on data in the past instead of the future, which aligns behavior with Prometheus. This resolves issues with empty last buckets that were previously encountered. ([PR 5366](https://github.com/grafana/tempo/pull/5366))

This change may cause differences in existing dashboards and alerts that rely on TraceQL metrics bucket calculations. Review your monitoring and alerting configurations after upgrading.

### Series label handling improvements

Fixed incorrect TraceQL metrics results when series labels include both strings and integers with the same textual representation (such as `"500"` vs `500`). The `prom_labels` field has been removed from TraceQL metrics responses as it was the source of these errors. ([PR 5659](https://github.com/grafana/tempo/pull/5659))

There may be brief interruptions to TraceQL metrics queries during rollout while components run different versions. Plan for a coordinated rollout to minimize impact.

### vParquet5 block format

Tempo 2.9 introduces a new experimental vParquet5 block format, designed to improve query performance and reduce storage requirements.vParquet5 has two previews: `vParquet5-preview1`, low-resolution timestamp columns, and `vParquet5-preview2`, dedicated integer columns.

Breaking changes are expected before the final release. ([PR 5495](https://github.com/grafana/tempo/pull/5495), [PR 5639](https://github.com/grafana/tempo/pull/5639))

### Go version upgrade

Tempo 2.9 upgrades to Go 1.25.0, which may affect custom builds or deployments with specific Go version requirements. ([PR 5548](https://github.com/grafana/tempo/pull/5548))

If you're building Tempo from source or using custom Docker images, ensure your build environment supports Go 1.25.0.

## Upgrade to Tempo 2.8

When upgrading to Tempo 2.8, be aware of these considerations and breaking changes.

### Changed the default listening port

With Tempo 2.8, the default `http_listen_port` changes from 80 to 3200. Check the configuration options for the `server:` block in your Tempo configuration file.

```yaml
server:
    # HTTP server listen host
    [http_listen_address: <string>]

    # HTTP server listen port
    [http_listen_port: <int> | default = 3200]
```

Refer to [issue 4945](https://github.com/grafana/tempo/discussions/4945) for more information for the rationale.
[PR 4960](https://github.com/grafana/tempo/pull/4960)

### Removed Tempo serverless

Tempo serverless has been removed. The following configuration options are no longer valid and should be removed from your Tempo configuration. [PR [4599](https://github.com/grafana/tempo/pull/4599/))

```yaml
querier:
  search:
    prefer_self: <int>
    external_hedge_requests_at: <duration>
    external_hedge_requests_up_to: <duration>
    external_backend: <string>
    google_cloud_run: <string>
    external_endpoints: <array>
```

In addition, these Tempo serverless related metrics have been removed: `tempo_querier_external_endpoint_duration_seconds`, `tempo_querier_external_endpoint_hedged_roundtrips_total`, and `tempo_feature_enabled`.

### Updated, removed, or renamed configuration parameters

| Parameter                                                         | Comments                                                                                                                                                                                                              |
| ----------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `max_span_attr_byte`                                              | Renamed to `max_attribute_bytes`. ([PR 4633](https://github.com/grafana/tempo/pull/4633))                                                                                                                             |
| `tempo_discarded_spans_total`                                     | Removed `internal_error` as a reason from `tempo_discarded_spans_total`. ([PR 4554](https://github.com/grafana/tempo/pull/4554))                                                                                      |
| `tempo_receiver_accepted_span` and `tempo_receiver_refused_spans` | The `name` dimension from `tempo_receiver_accepted_span` and `tempo_receiver_refused_spans` changes from `tempo/jaeger_receiver` to `jaeger/jaeger_receiver`. ([PR 4893](https://github.com/grafana/tempo/pull/4893)) |

### Other upgrade considerations

- Upgrade OTEL Collector to v0.122.1. The `name` dimension from `tempo_receiver_accepted_span` and `tempo_receiver_refused_spans` changes from `tempo/jaeger_receiver` to `jaeger/jaeger_receiver`. ([PR 4893](https://github.com/grafana/tempo/pull/4893))
- Replace `opentracing-contrib/go-grpc` by `otelgrpc` in Tempo query. ([PR 4958](https://github.com/grafana/tempo/pull/4958))
- Enforce max attribute size at event, link, and instrumentation scope. The configuration is now per-tenant. Renamed `max_span_attr_byte` to `max_attribute_bytes`. ([PR 4633](https://github.com/grafana/tempo/pull/4633))
- Converted SLO metric `query_frontend_bytes_processed_per_second` from a histogram to a counter as it's more performant. ([PR 4748](https://github.com/grafana/tempo/pull/4748))
- Removed the OpenTelemetry Jaeger exporter, which has been [deprecated](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/jaeger). ([PR 4926](https://github.com/grafana/tempo/pull/4926))

## Upgrade to Tempo 2.7

When upgrading to Tempo 2.7, be aware of these considerations and breaking changes.

### OpenTelemetry Collector receiver listens on `localhost` by default

After this change, the OpenTelemetry Collector receiver defaults to binding on `localhost` rather than `0.0.0.0`. Tempo installations running in Docker or other container environments must update their listener address to continue receiving data. ([#4465](https://github.com/grafana/tempo/pull/4465))

Most Tempo installations use the receivers with the default configuration:

```yaml
distributor:
  receivers:
    otlp:
      protocols:
        grpc:
        http:
```

This used to work fine since the receivers defaulted to `0.0.0.0:4317` and `0.0.0.0:4318` respectively. With the changes to replace unspecified addresses, the receivers now default to `localhost:4317` and `localhost:4318`.

As a result, connections to Tempo running in a Docker container won't work anymore.

To workaround this, you need to specify the address you want to bind to explicitly. For instance, if Tempo is running in a container with hostname `tempo`, this should work:

```yaml
# ...
http:
  endpoint: 'tempo:4318'
```

You can also explicitly bind to `0.0.0.0` still, but this has potential security risks:

```yaml
# ...
http:
  endpoint: '0.0.0.0:4318'
```

### Maximum spans per span set

A new `max_spans_per_span_set` limit is enabled by default and set to 100.
Set it to `0` to restore the old behavior (unlimited).
Otherwise, spans beyond the configured max are dropped. ([#4275](https://github.com/grafana/tempo/pull/4383))

```
query_frontend:
  search:
      max_spans_per_span_set: 0
```

### Tempo serverless deprecation

Tempo serverless is officially deprecated and will be removed in an upcoming release.
Prepare to migrate any serverless workflows to alternative deployments. ([#4017](https://github.com/grafana/tempo/pull/4017))

There are no changes to this release for serverless. However, you'll need to remove these configurations before the next release.

### Anchored regular expressions matchers in TraceQL

Regex matchers in TraceQL are now fully anchored using Prometheus's fast regexp.
For instance, `span.foo =~ "bar"` is interpreted as `span.foo =~ "^bar$"`. Adjust existing queries accordingly. ([#4329](https://github.com/grafana/tempo/pull/4329))

For more information, refer to the [Comparison operators TraceQL](http://localhost:3002/docs/tempo/<TEMPO_VERSION>/traceql/construct-traceql-queries/#comparison-operators) documentation.

### Migration from OpenTracing to OpenTelemetry

The `use_otel_tracer` option is removed.
Configure your spans via standard OpenTelemetry environment variables.
For Jaeger exporting, set `OTEL_TRACES_EXPORTER=jaeger`.For more information, refer to the [OpenTelemetry documentation](https://www.google.com/url?q=https://opentelemetry.io/docs/languages/sdk-configuration/&sa=D&source=docs&ust=1736460391410238&usg=AOvVaw3bykVWwn34XfhrnFK73uM_). ([#3646](https://github.com/grafana/tempo/pull/3646))

### gRPC compression set to snappy

Tempo 2.7.1 set gRPC compression between all components to be `snappy`.
Using `snappy` provides a balanced approach to compression between components that will work for most installations.

If you prefer a different balance of CPU/Memory and bandwidth, consider disabling compression or using zstd.

For a discussion on alternatives, refer to [this discussion thread](https://github.com/grafana/tempo/discussions/4683). ([#4696](https://github.com/grafana/tempo/pull/4696)).

#### gRPC compression disabled

Tempo 2.7.0 release disabled gRPC compression in the querier and distributor for performance reasons. ([#4429](https://github.com/grafana/tempo/pull/4429)).
Our benchmark suggests that without compression, queriers and distributors use less CPU and memory.

Tempo 2.7.1 changed the default value to `snappy` for internal components.

{{< admonition type="note" >}}
This change may increase data usage and network traffic, which can impact cloud billing.
{{< /admonition >}}

If you notice increased network traffic or issues, check the gRPC compression settings.

For instructions how to enable gRPC compression, refer to [gRPC compression configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#grpc-compression) for more information.

### Added, updated, removed, or renamed configuration parameters

<table>
  <tr>
   <td><strong>Parameter</strong>
   </td>
   <td><strong>Comments</strong>
   </td>
  </tr>
  <tr>
   <td><code>querier_forget_delay</code>
   </td>
   <td>Removed. The <code>querier_forget_delay</code> setting provided no effective functionality and has been dropped. (<a href="https://github.com/grafana/tempo/pull/3996">#3996</a>)
   </td>
  </tr>
  <tr>
   <td><code>use_otel_tracer</code>
   </td>
   <td>Removed. Configure your spans via standard OpenTelemetry environment variables. For Jaeger exporting, set <code>OTEL_TRACES_EXPORTER=jaeger</code>. (<a href="https://github.com/grafana/tempo/pull/3646">#3646</a>)
   </td>
  </tr>
  <tr>
   <td><code>max_spans_per_span_set</code>
   </td>
   <td>Added to query-frontend configuration. (<a href="https://github.com/grafana/tempo/pull/4383">#4275</a>)
   </td>
  </tr>
  <tr>
   <td><code>use_otel_tracer</code>
   </td>
   <td>The <code>use_otel_tracer</code> option is removed. Configure your spans via standard OpenTelemetry environment variables. For Jaeger exporting, set <code>OTEL_TRACES_EXPORTER=jaeger</code>. (<a href="https://github.com/grafana/tempo/pull/3646">#3646</a>)
   </td>
  </tr>
</table>

### Other upgrade considerations

- The Tempo CLI now targets the `/api/v2/traces` endpoint by default. Use the `--v1` flag if you still rely on the older `/api/traces` endpoint. ([#4127](https://github.com/grafana/tempo/pull/4127))
- If you already set the `X-Scope-OrgID` header in per-tenant overrides or global Tempo configuration, it is now honored and not overwritten by Tempo. This may change behavior if you previously depended on automatic injection. ([#4021](https://github.com/grafana/tempo/pull/4021))
- The AWS Lambda build output changes from main to bootstrap. Follow the [AWS migration steps](https://aws.amazon.com/blogs/compute/migrating-aws-lambda-functions-from-the-go1-x-runtime-to-the-custom-runtime-on-amazon-linux-2/) to ensure your Lambda functions continue to work. ([#3852](https://github.com/grafana/tempo/pull/3852))

## Upgrade to Tempo 2.6

Tempo 2.6 has several considerations for any upgrade:

- Operational change for TraceQL metrics
- vParquet4 is now the default block format
- Updated, removed, or renamed parameters

For a complete list of changes, refer to the [Tempo 2.6 CHANGELOG](https://github.com/grafana/tempo/releases/tag/v2.6.0).

### Operational change for TraceQL metrics

We've changed to an RF1 (Replication Factor 1) pattern for TraceQL metrics as we were unable to hit performance goals for RF3 deduplication. This requires some operational changes to query TraceQL metrics.

TraceQL metrics are still considered experimental, but we hope to mark them GA soon when we productionize a complete RF1 write-read path. [PRs [3628](https://github.com/grafana/tempo/pull/3628), [3691](https://github.com/grafana/tempo/pull/3691), [3723](https://github.com/grafana/tempo/pull/3723), [3995](https://github.com/grafana/tempo/pull/3995)]

**For recent data**

The local-blocks processor must be enabled to start using metrics queries like `{ } | rate()`. If not enabled metrics queries fail with the error `localblocks processor not found`. Enabling the local-blocks processor can be done either per tenant or in all tenants.

- Per-tenant in the per-tenant overrides:

  ```yaml
  overrides:
    'tenantID':
      metrics_generator_processors:
        - local-blocks
  ```

- By default, for all tenants in the main configuration:

  ```yaml
  overrides:
    defaults:
      metrics_generator:
        processors: [local-blocks]
  ```

Add this configuration to run TraceQL metrics queries against all spans (and not just server spans):

```yaml
metrics_generator:
  processor:
    local_blocks:
      filter_server_spans: false
```

**For historical data**

To run metrics queries on historical data, you must configure the local-blocks processor to flush RF1 blocks to object storage:

```yaml
metrics_generator:
  processor:
    local_blocks:
      flush_to_storage: true
```

### Transition to vParquet4

vParquet4 format is now the default block format.
It's production ready and we highly recommend switching to it for improved query performance. [PR [3810](https://github.com/grafana/tempo/pull/3810)]

Upgrading to Tempo 2.6 modifies the Parquet block format.
You don't need to do anything with Parquet to go from 2.5 to 2.6.
If you used vParquet2 or vParquet3, all of your old blocks remain and can be read by Tempo 2.6.
Tempo 2.6 creates vParquet4 blocks by default, which enables the new TraceQL features.

Although you can use Tempo 2.6 with vParquet2 or vParquet3, you can only use vParquet4 with Tempo 2.5 and later.
If you are using 2.5 with vParquet4, you'll need to upgrade to Tempo 2.6 to use the new TraceQL features.

You can also use the `tempo-cli analyse blocks` command to query vParquet4 blocks. [PR 3868](https://github.com/grafana/tempo/pull/3868)]
Refer to the [Tempo CLI](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/tempo_cli/#analyse-blocks)documentation for more information.

For information on upgrading, refer to [Choose a different block format](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/parquet/#choose-a-different-block-format).

### Updated, removed, or renamed configuration parameters

<table>
  <tr>
   <td>Parameter
   </td>
   <td>Comments
   </td>
  </tr>
  <tr>
   <td><code>storage:</code>
<p>
<code>  azure:</code>
<p>
<code>    use_v2_sdk: </code>
   </td>
   <td>Removed. Azure v2 is the only and primary Azure backend [PR <a href="https://github.com/grafana/tempo/pull/3875">#3875</a>]
   </td>
  </tr>
  <tr>
   <td><code>autocomplete_filtering_enabled</code>
   </td>
   <td>The feature flag option has been removed. The feature is always enabled. [PR  <a href="https://github.com/grafana/tempo/pull/3729">#3729</a>]
   </td>
  </tr>
  <tr>
   <td><code>completedfilepath</code> and <code>blocksfilepath</code>
   </td>
   <td>Removed unused WAL configuration options. [PR <a href="https://github.com/grafana/tempo/pull/3911">#3911</a>]
   </td>
  </tr>
  <tr>
   <td><code>compaction_disabled</code>
   </td>
   <td>New. Allow compaction disablement per-tenant. [PR <a href="https://github.com/grafana/tempo/pull/3965">#3965</a>, <a href="https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#overrides">documentation</a>]
   </td>
  </tr>
  <tr>
   <td><code>Storage:</code>
<p>
<code>  s3:</code>
<p>
<code>    [enable_dual_stack: &lt;bool>]</code>
   </td>
   <td>Boolean flag to activate or deactivate <a href="https://docs.aws.amazon.com/AmazonS3/latest/userguide/dual-stack-endpoints.html">dualstack mode</a> on the Storage block configuration for S3. [PR <a href="https://github.com/grafana/tempo/pull/3721">#3721</a>, <a href="https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#standard-overrides">documentation</a>]
   </td>
  </tr>
</table>

### `tempo-query` is a standalone server

With Tempo 2.6.1, `tempo-query` is no longer a Jaeger instance with `grpcPlugin`.
It's now a standalone server.
Serving a gRPC API for Jaeger on 0.0.0.0:7777 by default. [PR 3840]

## Upgrade to Tempo 2.5

Tempo 2.5 has several considerations for any upgrade:

- Docker image runs as a new UID
- Support for vParquet format removed
- Experimental vParquet4 block format
- Removed configuration parameters

For a complete list of changes, enhancements, and bug fixes, refer to the [Tempo 2.5 CHANGELOG](https://github.com/grafana/tempo/releases/tag/v2.5.0).

### Docker image runs as new UID

The Tempo process in the [official Docker image](https://hub.docker.com/r/grafana/tempo/tags) used to run as `root`. The Tempo process now runs as UID `10001` in the Docker image.

Components such as ingesters and metrics generators that maintain files on disk won't come up cleanly without intervention.
The new user `10001` won't have access to the old files created by `root`.

The ownership of `/var/tempo` changed from `root:root` to `tempo:tempo` with the UID/GID of `10001`.

The `ingester` and `metrics-generator` statefulsets may need to [run `chown`](https://opensource.com/article/19/8/linux-chown-command) to change ownership to start properly.

Refer to [PR 2265](https://github.com/grafana/tempo/pull/2265) to see a Jsonnet example of an `init` container.

This change doesn't impact you if you used the Helm chart with the default security context set in the chart.
All data should be owned by the `tempo` user already.
The UID won't impact Helm chart users.

<!-- vale Grafana.Spelling = NO -->

### Support for vParquet format removed

The original vParquet format [has been removed](https://github.com/grafana/tempo/pull/3663) from Tempo 2.5.
Direct upgrades from Tempo 2.1 to Tempo 2.5 aren't possible.
You will need to upgrade to an intermediate version and wait for the old vParquet blocks to fall out of retention before upgrading to 2.5. [PR 3663](https://github.com/grafana/tempo/pull/3663)]

vParquet(1) won't be recognized as a valid encoding and any remaining vParquet(1) blocks won't be readable.

Installations running with historical defaults should not require any changes as the default has been migrated for several releases.
Installations with storage settings pinned to vParquet must run a previous release configured for vParquet2 or higher until all existing vParquet(1) blocks have expired and been deleted from the backend, or else you'll encounter read errors after upgrading to this release.

### Experimental vParquet4 block format

The vParquet4 block format is required for querying links, events, and arrays and improves query performance relative to previous formats. vParquet4 will be the default block format in the next release. [[PR 3368](https://github.com/grafana/tempo/pull/3368)]

While you can use vParquet4, keep in mind that it's experimental.
If you choose to use vParquet4 and then opt to revert to vParquet3, any vParquet4 blocks would not be readable by vParquet3.

To try vParquet4, refer to [Choose a block format](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/parquet/#choose-a-different-block-format).

<!-- vale Grafana.Spelling = YES -->

### Removed configuration parameters

<table>
 <tr>
  <td><strong>Parameter</strong>
  </td>
  <td><strong>Comments</strong>
  </td>
 </tr>
 <tr>
  <td>`[hedge_requests_at: &lt;duration> | default = 2s ]`
<p>
`[hedge_requests_up_to: &lt;int> | default = 2 ]`
  </td>
  <td>Removed options from the configuration. [PR <a href="https://github.com/grafana/tempo/pull/3522">#3522</a>]
  </td>
 </tr>
</table>

### Additional considerations

- Updating to OTLP 1.3.0 removes the deprecated `InstrumentationLibrary` and `InstrumentationLibrarySpan` from the OTLP receivers. [PR 3649](https://github.com/grafana/tempo/pull/3649)]
- Removes the addition of a tenant in multi-tenant trace id lookup. [PR 3522](https://github.com/grafana/tempo/pull/3522)]

## Upgrade to Tempo 2.4

Tempo 2.4 has several considerations for any upgrade:

<!-- vale Grafana.Spelling = NO -->

- vParquet3 is now the default backend
- Caches configuration was refactored
- Updated, removed, and renamed configuration parameters

For a complete list of changes, enhancements, and bug fixes, refer to the [Tempo 2.4 CHANGELOG](https://github.com/grafana/tempo/releases).

### Transition to vParquet3 as default block format

vParquet3 format is now the default block format. It is production ready and we highly recommend switching to it for improved query performance and [dedicated attribute columns](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/dedicated_columns/).

Upgrading to Tempo 2.4 modifies the Parquet block format. Although you can use Tempo 2.3 with vParquet2 or vParquet3, you can only use Tempo 2.4 with vParquet3.

With this release, the first version of our Parquet backend, vParquet, is being deprecated.
Tempo 2.4 still reads vParquet1 blocks.
However, Tempo will exit with error if they are manually configured. [[PR 3377](https://github.com/grafana/tempo/pull/3377/files#top)]

For information on changing the vParquet version, refer to [Choose a different block format](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/parquet#choose-a-different-block-format).

<!-- vale Grafana.Spelling = YES -->

### Cache configuration refactored

The major cache refactor lets you configure multiple role-based caches. [[PR 3166](https://github.com/grafana/tempo/pull/3166)]
This change resulted in several fields being deprecated (refer to the old configuration).

These fields have all been migrated to a top level `cache:` field.

For more information about the configuration, refer to the [Cache](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#cache) section.

The old configuration block looked like this:

```yaml
storage:
  trace:
    cache:
    search:
      cache_control:
    background_cache:
    memcached:
    redis:
```

With the new configuration, you create your list of caches, with either `redis` or `memcached` cluster with your configuration, and then define the types of data and roles.

Simple configuration example:

```yaml
cache:
  caches:
    - memcached:
        host: <some memcached cluster>
      roles:
        - bloom
        - parquet-footer
    - memcached:
        host: <some memcached cluster>
      roles:
        - frontend-search
```

### Updated, removed, or renamed configuration parameters

<table>
  <tr>
   <td>Parameter
   </td
   <td>Comments
   </td>
  </tr>
  <tr>
   <td><code>distributor.log_received_traces</code>
   </td>
   <td>Use the <code>distributor.log_received_spans</code> configuration block instead. [PR <a href="https://github.com/grafana/tempo/pull/3008">#3008</a>]
   </td>
  </tr>
  <tr>
   <td><code>tempo_query_frontend_queries_total{op="searchtags|metrics"}</code>
   </td>
   <td>Removed deprecated frontend metrics configuration option
   </td>
  </tr>
</table>

The distributor now returns 200 for any batch containing only `trace_too_large` and `max_live_traces` errors. The number of discarded spans are still reflected in the `tempo_discarded_spans_total metrics`.

## Upgrade to Tempo 2.3

Tempo 2.3 has several considerations for any upgrade:

- vParquet3 is available as a stable, production-read block format
- Configuration option to use Azure SDK v2
- New `defaults` block in Overrides module configuration
- Several configuration parameters have been renamed or removed.

For a complete list of changes, enhancements, and bug fixes, refer to the [Tempo 2.3 CHANGELOG](https://github.com/grafana/tempo/releases).

### Production-ready vParquet3 block format

vParquet3 provides improved query performance and [dedicated attribute columns](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/dedicated_columns/).

This block format is required for using dedicated attribute columns.

While vParquet2 remains the default backend for Tempo 2.3, vParquet3 is available as a stable option.
Both work with Tempo 2.3.

Upgrading to Tempo 2.3 doesn't modify the Parquet block format.

{{< admonition type="note" >}}
Tempo 2.2 can't read data stored in vParquet3.
{{< /admonition >}}

Recommended update process:

1. Upgrade your Tempo installation to version 2.3, remaining on vParquet2.
2. Verify the upgrade is stable and performs as expected. If you notice any issues, you can downgrade to version 2.2, and data remains readable.
3. [Change the block format to vParquet3](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/parquet/).

If you notice any issues on step 3 using the new block format, you can downgrade to vParquet2.
All your data remains readable in Tempo 2.3.
However, if you have vParquet3 blocks and have to downgrade to Tempo 2.2, you will have data loss.

### Use Azure SDK v2

If you are using Azure storage, we recommend using the v2 SDK, [azure-sdk-for-go](https://github.com/Azure/azure-sdk-for-go).
You can use the `use_v2_sdk` configure option for switching.

For more information, refer to the [Storage block configuration example documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#storage-block-configuration-example).

### New `defaults` block in Overrides module configuration

The Overrides module has a new `defaults` block for configuring global or per-tenant settings.
The Overrides format now includes changes to indented syntax.
For more information, read the [Overrides configuration documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#overrides).

You can also use the Tempo CLI to migrate configurations. Refer to the [tempo-cli documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/tempo_cli/#migrate-overrides-config-command).

The old configuration block looked like this:

```yaml
overrides:
  ingestion_rate_strategy: local
  ingestion_rate_limit_bytes: 12345
  ingestion_burst_size_bytes: 67890
  max_search_duration: 17s
  forwarders: ['foo']
  metrics_generator_processors: [service-graphs, span-metrics]
```

The new configuration block looks like this:

```yaml
overrides:
  defaults:
    ingestion:
      rate_strategy: local
      rate_limit_bytes: 12345
      burst_size_bytes: 67890
    read:
      max_search_duration: 17s
    forwarders: ['foo']
    metrics_generator:
      processors: [service-graphs, span-metrics]
```

### Removed or renamed configuration parameters

<table>
  <tr>
   <td><strong>Parameter</strong>
   </td>
   <td><strong>Comments</strong>
   </td>
  </tr>
  <tr>
   <td><code>distributor.log_received_traces</code>
   </td>
   <td>Use the <code>distributor.log_received_spans</code> configuration block instead. [PR <a href="https://github.com/grafana/tempo/pull/3008">3008</a>]
   </td>
  </tr>
  <tr>
   <td><code>tempo_query_frontend_queries_total{op="searchtags|metrics"}</code>
   </td>
   <td>Removed deprecated frontend metrics configuration option
   </td>
  </tr>
</table>

## Upgrade to Tempo 2.2

Tempo 2.2 has several considerations for any upgrade:

- vParquet2 is now the default block format
- Several configuration parameters have been renamed or removed.

For a complete list of changes, enhancements, and bug fixes, refer to the [Tempo 2.2 CHANGELOG](https://github.com/grafana/tempo/releases).

### Default block format changed to vParquet2

While not a breaking change, upgrading to Tempo 2.2 by default changes Tempo's block format to vParquet2.

To stay on a previous block format, read the [Parquet configuration documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/parquet/#choose-a-different-block-format).
We strongly encourage upgrading to vParquet2 as soon as possible as this is required for using structural operators in your TraceQL queries and provides query performance improvements, in particular on queries using the `duration` intrinsic.

### Updated JSonnet supports `statefulset` for the metrics-generator

Tempo 2.2 updates the `microservices` JSonnet to support a `statefulset` for the `metrics_generator` component.

{{< admonition type="note" >}}
This update is important if you use the experimental `local-blocks` processor.
{{< /admonition >}}

To support a new `processor`, the metrics-generator has been converted from a `deployment` into a `statefulset` with a PVC.
This requires manual intervention to migrate successfully and avoid downtime.
Note that currently both a `deployment` and a `statefulset` will be managed by the JSonnet for a period of time, after which we will delete the deployment from this repository and you will need to delete user-side references to the `tempo_metrics_generator_deployment`, as well as delete the deployment itself.

Refer to the PR for seamless migration instructions. [PRs [2533](https://github.com/grafana/tempo/pull/2533), [2467](https://github.com/grafana/tempo/pull/2467)]

### Removed or renamed configuration parameters

The following fields were removed or renamed.
| Parameter | Comments |
|---|---|
|<pre>query_frontend:<br>&nbsp;&nbsp;tolerate_failed_blocks: <int></pre> | Remove support for `tolerant_failed_blocks` [[PR 2416](https://github.com/grafana/tempo/pull/2416)] |
|<pre>storage:<br>&nbsp;&nbsp;trace:<br>&nbsp;&nbsp;s3:<br>&nbsp;&nbsp;insecure_skip_verify: true // renamed to tls_insecure_skip_verify</pre> | Renamed `insecure_skip_verify` to `tls_insecure_skip_verify` [[PR 2407](https://github.com/grafana/tempo/pull/2407)] |

## Upgrade to Tempo 2.1

Tempo 2.1 has two major considerations for any upgrade:

- Support for search on v2 block is removed
- Breaking changes to metric names

For more information on other enhancements, read the [Tempo 2.1 release notes](https://grafana.com/docs/tempo/<TEMPO_VERSION>/release-notes/v2-1/).

### Remove support for Search on v2 blocks

Users can no longer search blocks in v2 format. Only the Parquet formats support search.
These search configuration options were removed from the overrides section:

```
overrides:
  max_search_bytes_per_trace:
  search_tags_allow_list:
  search_tags_deny_list:
```

The following metrics configuration was also removed:

```
tempo_ingester_trace_search_bytes_discarded_total
```

### Upgrade path to maintain search from Tempo 1.x to 2.1

<!-- vale Grafana.Spelling = NO -->

Removing support for search on v2 blocks means that if you upgrade directly from 1.9 to 2.1, you will not be able to search your v2 blocks. To avoid this, upgrade to 2.0 first, since 2.0 supports searching both v2 and vParquet blocks. You can let your old v2 blocks gradually age out while Tempo creates new vParquet blocks from incoming traces. Once all of your v2 blocks have been deleted and you only have vParquet format-blocks, you can upgrade to Tempo 2.1. All of your blocks will be searchable.

<!-- vale Grafana.Spelling = YES -->

Parquet files are no longer cached when carrying out searches.

### Breaking changes to metric names exposed by Tempo

All Prometheus metrics exposed by Tempo on its `/metrics` endpoint that were previously prefixed with `cortex_` have now been renamed to be prefixed with `tempo_` instead. (PR [2204](https://github.com/grafana/tempo/pull/2204))

Tempo now includes SLO metrics to count where queries are returned within a configurable time range. (PR [2008](https://github.com/grafana/tempo/pull/2008))

The `query_frontend_result_metrics_inspected_bytes` metric was removed in favor of `query_frontend_bytes_processed_per_second`.

## Upgrade from Tempo 1.5 to 2.0

Tempo 2.0 marks a major milestone in Tempo development. When planning your upgrade, consider these factors:

- Breaking changes:
  - Renamed, removed, and moved configurations are described in section below.
  - The `TempoRequestErrors` alert was removed from mixin. Any Jsonnet users relying on this alert should copy this into their own environment.
- Advisory:
  - Changed defaults. Are these updates relevant for your installation?
  - TraceQL editor needs to be enabled in Grafana to use the query editor.
  - Resource requirements have changed for Tempo 2.0 with the default configuration.

Once you upgrade to Tempo 2.0, there is no path to downgrade.

{{< admonition type="note" >}}
There is a potential issue loading experimental Parquet storage blocks. You may see errors or even panics in the compactors. We have only been able to reproduce this with interim commits between 1.5 and 2.0, but if you experience any issues, [report them](https://github.com/grafana/tempo/issues/new?assignees=&labels=&template=bug_report.md&title=).
{{< /admonition >}}

### Check Tempo installation resource allocation

Parquet provides faster search and is required to enable TraceQL. However, the Tempo installation requires additional CPU and memory resources to use Parquet efficiently. Parquet is more costly due to the extra work of building the columnar blocks, and operators should expect at least 1.5x increase in required resources to run a Tempo 2.0 cluster. Most users find these extra resources are negligible compared to the benefits that come from the additional features of TraceQL and from storing traces in an open format.

You can continue using the previous `v2` block format using the instructions provided in the [Parquet configuration documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/parquet/).
Tempo continues to support trace by id lookup on the `v2` format for the foreseeable future.

### Enable TraceQL in Grafana

TraceQL is enabled by default in Tempo 2.0. The TraceQL query editor requires Grafana 9.3.2 and later.

The TraceQL query editor is in beta in Grafana 9.3.2 and needs to be enabled with the `traceqlEditor` feature flag.

### Check configuration options for removed and renamed options

The following tables describe the parameters that have been removed or renamed.

#### Removed and replaced

| Parameter                                                  | Comments                                                                                    |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| <pre>query_frontend:<br>&nbsp;&nbsp;query_shards:</pre>    | Replaced by `trace_by_id.query_shards`.                                                     |
| <pre>querier:<br>&nbsp;&nbsp;query_timeout:</pre>          | Replaced by two different settings: `search.query_timeout` and `trace_by_id.query_timeout`. |
| <pre>ingester:<br>&nbsp;&nbsp;use_flatbuffer_search:</pre> | Removed and automatically determined based on block format.                                 |
| `search_enabled`                                           | Removed. Now defaults to true.                                                              |
| `metrics_generator_enabled`                                | Removed. Now defaults to true.                                                              |

#### Renamed

The following `compactor` configuration parameters were renamed.

| Parameter                                                   | Comments                              |
| ----------------------------------------------------------- | ------------------------------------- |
| <pre>compaction:<br>&nbsp;&nbsp;chunk_size_bytes:</pre>     | Renamed to `v2_in_buffer_bytes`       |
| <pre>compaction:<br>&nbsp;&nbsp;flush_size_bytes:</pre>     | Renamed to `v2_out_buffer_bytes`      |
| <pre>compaction:<br>&nbsp;&nbsp;iterator_buffer_size:</pre> | Renamed to `v2_prefetch_traces_count` |

The following `storage` configuration parameters were renamed.

| Parameter                                                | Comments                                  |
| -------------------------------------------------------- | ----------------------------------------- |
| <pre>wal:<br>&nbsp;&nbsp;encoding:</pre>                 | Renamed to `v2_encoding`                  |
| <pre>block:<br>&nbsp;&nbsp;index_downsample_bytes:</pre> | Renamed to `v2_index_downsample_bytes`    |
| <pre>block:<br>&nbsp;&nbsp;index_page_size_bytes:</pre>  | Renamed to `v2_index_page_size_bytes`     |
| <pre>block:<br>&nbsp;&nbsp;encoding:</pre>               | Renamed to `v2_encoding`                  |
| <pre>block:<br>&nbsp;&nbsp;row_group_size_bytes:</pre>   | Renamed to `parquet_row_group_size_bytes` |

The Azure Storage configuration section now uses snake case with underscores (`_`) instead of dashes (`-`). Example of using snake case on Azure Storage config:

```yaml
# config.yaml
storage:
  trace:
    azure:
      storage_account_name:
      storage_account_key:
      container_name:
```

<!-- vale Grafana.We = YES -->
<!-- vale Grafana.Will = YES -->
<!-- vale Grafana.Timeless = YES -->
