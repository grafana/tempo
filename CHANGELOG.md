## main / unreleased
* [ENHANCEMENT] Add support for searching by span status message using  `statusMessage` keyword [#2848](https://github.com/grafana/tempo/pull/2848) (@kousikmitra)
* [FEATURE] Add the `/api/status/buildinfo` endpoint [#2702](https://github.com/grafana/tempo/pull/2702) (@fabrizio-grafana)
* [FEATURE] New encoding vParquet3 with support for dedicated attribute columns (@mapno, @stoewer) [#2649](https://github.com/grafana/tempo/pull/2649)
* [FEATURE] Add filtering support to Generic Forwarding [#2742](https://github.com/grafana/tempo/pull/2742) (@Blinkuu)
* [FEATURE] Add cli command to print out summary of large traces [#2775](https://github.com/grafana/tempo/pull/2775) (@ie-pham)
* [CHANGE] Update Go to 1.21 [#2486](https://github.com/grafana/tempo/pull/2829) (@zalegrala)
* [CHANGE] Make metrics-generator ingestion slack per tenant [#2589](https://github.com/grafana/tempo/pull/2589) (@ie-pham)
* [ENHANCEMENT] Add block indexes to vParquet2 and vParquet3 to improve trace by ID lookup [#2697](https://github.com/grafana/tempo/pull/2697) (@mdisibio)
* [ENHANCEMENT] Assert ingestion rate limits as early as possible [#2640](https://github.com/grafana/tempo/pull/2703) (@mghildiy)
* [ENHANCEMENT] Add several metrics-generator fields to user-configurable overrides [#2711](https://github.com/grafana/tempo/pull/2711) (@kvrhdn)
* [ENHANCEMENT] Update /api/metrics/summary to correctly handle missing attributes and improve performance of TraceQL `select()` queries. [#2765](https://github.com/grafana/tempo/pull/2765) (@mdisibio)
* [ENHANCEMENT] Tempo CLI command to convert from vParquet2 -> 3. [#2828](https://github.com/grafana/tempo/pull/2828) (@joe-elliott)
* [ENHANCEMENT] Add `TempoUserConfigurableOverridesReloadFailing` alert [#2784](https://github.com/grafana/tempo/pull/2784) (@kvrhdn)
* [ENHANCEMENT] Add RootSpanName and RootServiceName to log about discarded spans [#2816](https://github.com/grafana/tempo/pull/2816) (@marcinginszt)
* [ENHANCEMENT] Add `UserID` to log message about rate limiting [#2850](https://github.com/grafana/tempo/pull/2850) (@lshippy)
* [BUGFIX] Fix panic in metrics summary api [#2738](https://github.com/grafana/tempo/pull/2738) (@mdisibio)
* [BUGFIX] Fix node role auth IDMSv1 [#2760](https://github.com/grafana/tempo/pull/2760) (@coufalja)
* [BUGFIX] Only search ingester blocks that fall within the request time range. [#2783](https://github.com/grafana/tempo/pull/2783) (@joe-elliott)
* [BUGFIX] Align tempo_query_frontend_queries_total and tempo_query_frontend_queries_within_slo_total. [#2840](https://github.com/grafana/tempo/pull/2840) (@joe-elliott)
* [BUGFIX] To support blob storage in Azure Stack Hub as backend. [#2853](https://github.com/grafana/tempo/pull/2853) (@chlislb)
  This query will now correctly tell you %age of requests that are within SLO:
  ```
  sum(rate(tempo_query_frontend_queries_within_slo_total{}[1m])) by (op) 
  /
  sum(rate(tempo_query_frontend_queries_total{}[1m])) by (op)
  ```
  **BREAKING CHANGE** Removed: tempo_query_frontend_queries_total{op="searchtags|metrics"}. 
* [CHANGE] Overrides module refactor [#2688](https://github.com/grafana/tempo/pull/2688) (@mapno)
    Added new `defaults` block to the overrides' module. Overrides change to indented syntax.
    Old config:
```
overrides:
ingestion_rate_strategy: local
ingestion_rate_limit_bytes: 12345
ingestion_burst_size_bytes: 67890
max_search_duration: 17s
forwarders: ['foo']
metrics_generator_processors: [service-graphs, span-metrics]
```
New config:
```
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

## v2.2.1 / 2023-08-??

* [BUGFIX] Fix incorrect metrics for index failures [#2781](https://github.com/grafana/tempo/pull/2781) (@zalegrala)
* [BUGFIX] Fix panic in the metrics-generator when using multiple tenants with default overrides [#2786](https://github.com/grafana/tempo/pull/2786) (@kvrhdn)
* [BUGFIX] Restore `tenant_header_key` removed in #2414. [#2795](https://github.com/grafana/tempo/pull/2795) (@joe-elliott)
* [BUGFIX] Disable streaming over http by default. [#2803](https://github.com/grafana/tempo/pull/2803) (@joe-elliott)

## v2.2.0 / 2023-07-31

* [CHANGE] Make vParquet2 the default block format [#2526](https://github.com/grafana/tempo/pull/2526) (@stoewer)
* [CHANGE] Disable tempo-query by default in Jsonnet libs. [#2462](https://github.com/grafana/tempo/pull/2462) (@electron0zero)
* [CHANGE] Integrate `gofumpt` into CI for formatting requirements [2584](https://github.com/grafana/tempo/pull/2584) (@zalegrala)
* [CHANGE] Change log level of two compactor messages from `debug` to `info`. [#2443](https://github.com/grafana/tempo/pull/2443) (@dylanguedes)
* [CHANGE] Remove `tenant_header_key` option from `tempo-query` config [#2414](https://github.com/grafana/tempo/pull/2414) (@kousikmitra)
* [CHANGE] **Breaking Change** Remove support tolerate_failed_blocks. [#2416](https://github.com/grafana/tempo/pull/2416) (@joe-elliott)
  Removed config option:
  ```
  query_frontend:
    tolerate_failed_blocks: <int>
  ```
* [CHANGE] Upgrade memcached version in jsonnet microservices [#2466](https://github.com/grafana/tempo/pull/2466) (@zalegrala)
* [CHANGE] Prefix service graph extra dimensions labels with `server_` and `client_` if `enable_client_server_prefix` is enabled [#2335](https://github.com/grafana/tempo/pull/2335) (@domasx2)
* [CHANGE] **Breaking Change** Rename s3.insecure_skip_verify [#2407](https://github.com/grafana/tempo/pull/2407) (@zalegrala)
```yaml
storage:
  trace:
    s3:
      insecure_skip_verify: true   // renamed to tls_insecure_skip_verify
```
* [CHANGE] Ignore context canceled errors in the queriers [#2440](https://github.com/grafana/tempo/pull/2440) (@joe-elliott)
* [CHANGE] Start flush queue worker after wal replay and block rediscovery [#2456](https://github.com/grafana/tempo/pull/2456) (@ie-pham)
* [CHANGE] Update Go to 1.20.4 [#2486](https://github.com/grafana/tempo/pull/2486) (@ie-pham)
* [CHANGE] **Breaking Change** Convert metrics generator from deployment to a statefulset in jsonnet. Refer to the PR for seamless migration instructions. [#2533](https://github.com/grafana/tempo/pull/2533) [#2467](https://github.com/grafana/tempo/pull/2647) (@zalegrala)
* [FEATURE] New experimental API to derive on-demand RED metrics grouped by any attribute, and new metrics generator processor [#2368](https://github.com/grafana/tempo/pull/2368) [#2418](https://github.com/grafana/tempo/pull/2418) [#2424](https://github.com/grafana/tempo/pull/2424) [#2442](https://github.com/grafana/tempo/pull/2442) [#2480](https://github.com/grafana/tempo/pull/2480) [#2481](https://github.com/grafana/tempo/pull/2481) [#2501](https://github.com/grafana/tempo/pull/2501) [#2579](https://github.com/grafana/tempo/pull/2579) [#2582](https://github.com/grafana/tempo/pull/2582) (@mdisibio @zalegrala)
* [FEATURE] New TraceQL structural operators descendant (>>), child (>), and sibling (~) [#2625](https://github.com/grafana/tempo/pull/2625) [#2660](https://github.com/grafana/tempo/pull/2660) (@mdisibio)
* [FEATURE] Add user-configurable overrides module [#2543](https://github.com/grafana/tempo/pull/2543) [#2682](https://github.com/grafana/tempo/pull/2682) [#2681](https://github.com/grafana/tempo/pull/2681) (@electron0zero @kvrhdn)
* [FEATURE] Add support for `q` query param in `/api/v2/search/<tag.name>/values` to filter results based on a TraceQL query [#2253](https://github.com/grafana/tempo/pull/2253) (@mapno)
To make use of filtering, configure `autocomplete_filtering_enabled`.
* [FEATURE] Add support for `by()` and `coalesce()` to TraceQL. [#2490](https://github.com/grafana/tempo/pull/2490) 
* [FEATURE] Add a GRPC streaming endpoint for traceql search [#2366](https://github.com/grafana/tempo/pull/2366) (@joe-elliott)
* [FEATURE] Add new API to summarize span metrics from generators [#2481](https://github.com/grafana/tempo/pull/2481) (@zalegrala)
* [FEATURE] Add `select()` to TraceQL [#2494](https://github.com/grafana/tempo/pull/2494) (@joe-elliott)
* [FEATURE] Add `traceDuration`, `rootName` and `rootServiceName` intrinsics to TraceQL [#2503](https://github.com/grafana/tempo/pull/2503) (@joe-elliott)
* [ENHANCEMENT] Add support for query batching between frontend and queriers to improve throughput [#2677](https://github.com/grafana/tempo/pull/2677) (@joe-elliott)
* [ENHANCEMENT] Add initial RBAC support for serverless backend queries, limited to Google CloudRun [#2487](https://github.com/grafana/tempo/pull/2593) (@modulitos)
* [ENHANCEMENT] Add capability to flush all remaining traces to backend when ingester is stopped [#2538](https://github.com/grafana/tempo/pull/2538)
* [ENHANCEMENT] Fill parent ID column and nested set columns [#2487](https://github.com/grafana/tempo/pull/2487) (@stoewer)
* [ENHANCEMENT] Add metrics generator config option to allow customizable ring port [#2399](https://github.com/grafana/tempo/pull/2399) (@mdisibio)
* [ENHANCEMENT] Improve performance of TraceQL regex [#2484](https://github.com/grafana/tempo/pull/2484) (@mdisibio)
* [ENHANCEMENT] log client ip to help identify which client is no org id [#2436](https://github.com/grafana/tempo/pull/2436)
* [ENHANCEMENT] Add `spss` parameter to `/api/search/tags`[#2308] to configure the spans per span set in response
* [ENHANCEMENT] Continue polling tenants on error with configurable threshold [#2540](https://github.com/grafana/tempo/pull/2540) (@mdisibio)
* [ENHANCEMENT] Fully skip over parquet row groups with no matches in the column dictionaries [#2676](https://github.com/grafana/tempo/pull/2676) (@mdisibio)
* [ENHANCEMENT] Add `prefix` configuration option to `storage.trace.azure` and `storage.trace.gcs` [#2362](https://github.com/grafana/tempo/pull/2386) (@kousikmitra)
* [ENHANCEMENT] Add support to filter using negated regex operator `!~` [#2410](https://github.com/grafana/tempo/pull/2410) (@kousikmitra)
* [ENHANCEMENT] Add `prefix` configuration option to `storage.trace.azure` and `storage.trace.gcs` [#2386](https://github.com/grafana/tempo/pull/2386) (@kousikmitra)
* [ENHANCEMENT] Add `prefix` configuration option to `storage.trace.s3` [#2362](https://github.com/grafana/tempo/pull/2362) (@kousikmitra)
* [ENHANCEMENT] Add support for `concurrent_shards` under `trace_by_id` [#2416](https://github.com/grafana/tempo/pull/2416) (@joe-elliott)
  ```
  query_frontend:
    trace_by_id:
      concurrent_shards: 3
  ```
* [ENHANCEMENT] Enable cross cluster querying by adding two config options. [#2598](https://github.com/grafana/tempo/pull/2598) (@joe-elliott)
  ```
  querier:
    secondary_ingester_ring: <string>
  metrics_generator:
    override_ring_key: <string>
  ```
* [ENHANCEMENT] Add `scope` parameter to `/api/search/tags` [#2282](https://github.com/grafana/tempo/pull/2282) (@joe-elliott)
  Create new endpoint `/api/v2/search/tags` that returns all tags organized by scope.
* [ENHANCEMENT] Ability to toggle off latency or count metrics in metrics-generator [#2070](https://github.com/grafana/tempo/pull/2070) (@AlexDHoffer)
* [ENHANCEMENT] Extend `/flush` to support flushing a single tenant [#2260](https://github.com/grafana/tempo/pull/2260) (@kvrhdn)
* [ENHANCEMENT] Add override to limit number of blocks inspected in tag value search [#2358](https://github.com/grafana/tempo/pull/2358) (@mapno)
* [ENHANCEMENT] New synchronous read mode for vParquet and vParquet2 [#2165](https://github.com/grafana/tempo/pull/2165) [#2535](https://github.com/grafana/tempo/pull/2535) (@mdisibio)
* [ENHANCEMENT] Add option to override metrics-generator ring port  [#2399](https://github.com/grafana/tempo/pull/2399) (@mdisibio)
* [ENHANCEMENT] Add support for IPv6 [#1555](https://github.com/grafana/tempo/pull/1555) (@zalegrala)
* [ENHANCEMENT] Add span filtering to spanmetrics processor [#2274](https://github.com/grafana/tempo/pull/2274) (@zalegrala)
* [ENHANCEMENT] Add ability to detect virtual nodes in the servicegraph processor [#2365](https://github.com/grafana/tempo/pull/2365) (@mapno)
* [ENHANCEMENT] Introduce `overrides.Interface` to decouple implementation from usage [#2482](https://github.com/grafana/tempo/pull/2482) (@kvrhdn)
* [ENHANCEMENT] Improve TraceQL throughput by asynchronously creating jobs [#2530](https://github.com/grafana/tempo/pull/2530) (@joe-elliott)
* [BUGFIX] Fix Search SLO by routing tags to a new handler. [#2468](https://github.com/grafana/tempo/issues/2468) (@electron0zero)
* [BUGFIX] tempodb integer divide by zero error [#2167](https://github.com/grafana/tempo/issues/2167) (@kroksys)
* [BUGFIX] metrics-generator: ensure Prometheus will scale up shards when remote write is lagging behind [#2463](https://github.com/grafana/tempo/issues/2463) (@kvrhdn)
* [BUGFIX] Fixes issue where matches and other spanset level attributes were not persisted to the TraceQL results. [#2490](https://github.com/grafana/tempo/pull/2490) 
* [BUGFIX] Fixes issue where ingester search could occasionally fail with file does not exist error [#2534](https://github.com/grafana/tempo/issues/2534) (@mdisibio)
* [BUGFIX] Tempo failed to find meta.json path after adding prefix in S3/GCS/Azure configuration. [#2585](https://github.com/grafana/tempo/issues/2585) (@WildCatFish)
* [BUGFIX] Delay logging config warnings until the logger has been initialized [#2645](https://github.com/grafana/tempo/pull/2645) (@kvrhdn)
* [BUGFIX] Fix issue where metrics-generator was setting wrong labels for traces_target_info [#2546](https://github.com/grafana/tempo/pull/2546) (@ie-pham)
* [FEATURE] Add `tempo-cli` commands `analyse block` and `analyse blocks` to analyse parquet blocks and output summaries of generic attribute columns [#2622](https://github.com/grafana/tempo/pull/2622) (@mapno)

## v2.1.1 / 2023-04-28
* [BUGFIX] Fix issue where Tempo sometimes flips booleans from false->true at storage time. [#2400](https://github.com/grafana/tempo/issues/2400) (@joe-elliott)

## v2.1.0 / 2023-04-26
* [CHANGE] Capture and update search metrics for TraceQL [#2087](https://github.com/grafana/tempo/pull/2087) (@electron0zero)
* [CHANGE] tempo-mixin: disable auto refresh every 10 seconds [#2290](https://github.com/grafana/tempo/pull/2290) (@electron0zero)
* [CHANGE] Update tempo-mixin to show request in Resources dashboard [#2281](https://github.com/grafana/tempo/pull/2281) (@electron0zero)
* [CHANGE] Add support for s3 session token in static config [#2093](https://github.com/grafana/tempo/pull/2093) (@farodin91)
* [CHANGE] **Breaking Change** Remove support for search on v2 blocks. [#2159](https://github.com/grafana/tempo/pull/2159) (@joe-elliott)
  Removed config options:
  ```
  overrides:
    max_search_bytes_per_trace:
    search_tags_allow_list:
    search_tags_deny_list:
  ```
  Removed metrics:
  `tempo_ingester_trace_search_bytes_discarded_total`
* [CHANGE] Stop caching parquet files for search [#2164](https://github.com/grafana/tempo/pull/2164) (@mapno)
* [CHANGE] Update Go to 1.20 [#2079](https://github.com/grafana/tempo/pull/2079) (@scalalang2)
* [CHANGE] **BREAKING CHANGE** Change metrics prefixed with `cortex_` to `tempo_` [#2204](https://github.com/grafana/tempo/pull/2204) (@mapno)
* [CHANGE] Upgrade OTel to v0.74.0 [#2317](https://github.com/grafana/tempo/pull/2317) (@mapno)
* [FEATURE] New parquet based block format vParquet2 [#2244](https://github.com/grafana/tempo/pull/2244) (@stoewer)
* [FEATURE] Add support for Azure Workload Identity authentication [#2195](https://github.com/grafana/tempo/pull/2195) (@LambArchie)
* [FEATURE] Add flag to check configuration [#2131](https://github.com/grafana/tempo/issues/2131) (@robertscherbarth @agrib-01)
* [FEATURE] Add flag to optionally enable all available Go runtime metrics [#2005](https://github.com/grafana/tempo/pull/2005) (@andreasgerstmayr)
* [FEATURE] Add support for span `kind` to TraceQL [#2217](https://github.com/grafana/tempo/pull/2217) (@joe-elliott)
* [FEATURE] Add support for min/max/avg aggregates to TraceQL[#2255](https://github.com/grafana/tempo/pull/2255) (@joe-elliott)
* [ENHANCEMENT] Add Throughput and SLO Metrics with SLOConfig in Query Frontend [#2008](https://github.com/grafana/tempo/pull/2008) (@electron0zero)
  - **BREAKING CHANGE** `query_frontend_result_metrics_inspected_bytes` metric removed in favour of `query_frontend_bytes_processed_per_second`
* [ENHANCEMENT] Metrics generator to make use of counters earlier [#2068](https://github.com/grafana/tempo/pull/2068) (@zalegrala)
* [ENHANCEMENT] Log when a trace is too large to compact [#2105](https://github.com/grafana/tempo/pull/2105) (@scalalang2)
* [ENHANCEMENT] Add support for arbitrary arithemtic to TraceQL queries [#2146](https://github.com/grafana/tempo/pull/2146) (@joe-elliott)
* [ENHANCEMENT] tempo-cli: add command to migrate a tenant [#2130](https://github.com/grafana/tempo/pull/2130) (@kvrhdn)
* [ENHANCEMENT] Added the ability to multiple span metrics by an attribute such as `X-SampleRatio` [#2172](https://github.com/grafana/tempo/pull/2172) (@altanozlu)
* [BUGFIX] Correctly connect context during compaction [#2220](https://github.com/grafana/tempo/pull/2220) (@ie-pham)
* [BUGFIX] Apply `rate()` to bytes/s panel in tenant's dashboard. [#2081](https://github.com/grafana/tempo/pull/2081) (@mapno)
* [BUGFIX] Retry copy operations during compaction in GCS backend [#2111](https://github.com/grafana/tempo/pull/2111) (@mapno)
* [BUGFIX] Fix float/int comparisons in TraceQL. [#2139](https://github.com/grafana/tempo/issues/2139) (@joe-elliott)
* [BUGFIX] Improve locking and search head block in SearchTagValuesV2 [#2164](https://github.com/grafana/tempo/pull/2164) (@mapno)
* [BUGFIX] Fix not closing WAL block file before attempting to delete the folder. [#2139](https://github.com/grafana/tempo/pull/2152) (@kostya9)
* [BUGFIX] Stop searching for virtual tags if there are any hits.
  This prevents invalid values from showing up for intrinsics like `status` [#2219](https://github.com/grafana/tempo/pull/2219) (@joe-elliott)
* [BUGFIX] Correctly return unique spans when &&ing and ||ing spansets. [#2254](https://github.com/grafana/tempo/pull/2254) (@joe-elliott)
* [BUGFIX] Support negative values on aggregate filters like `count() > -1`. [#2289](https://github.com/grafana/tempo/pull/2289) (@joe-elliott)
* [BUGFIX] Support float as duration like `{duration > 1.5s}` [#2304](https://github.com/grafana/tempo/pull/2304) (@ie-pham)
* [ENHANCEMENT] Supports range operators for strings in TraceQL [#2321](https://github.com/grafana/tempo/pull/2321) (@ie-pham)
* [ENHANCEMENT] Supports TraceQL in Vulture [#2321](https://github.com/grafana/tempo/pull/2321) (@ie-pham)
* [FEATURE] Add job & instance labels to span metrics, a new target_info metrics, and custom dimension label mapping [#2261](https://github.com/grafana/tempo/pull/2261) (@ie-pham)

## v2.0.1 / 2023-03-03

* [CHANGE] No longer return `status.code` from /api/search/tags unless it is an attribute present in the data [#2059](https://github.com/grafana/tempo/issues/2059) (@mdisibio)
* [BUGFIX] Suppress logspam in single binary mode when metrics generator is disabled. [#2058](https://github.com/grafana/tempo/pull/2058) (@joe-elliott)
* [BUGFIX] Error more gracefully while reading some blocks written by an interim commit between 1.5 and 2.0 [#2055](https://github.com/grafana/tempo/pull/2055) (@mdisibio)
* [BUGFIX] Correctly coalesce trace level data when combining Parquet traces. [#2095](https://github.com/grafana/tempo/pull/2095) (@joe-elliott)
* [BUGFIX] Unescape query parameters in AWS Lambda to allow TraceQL queries to work. [#2114](https://github.com/grafana/tempo/issues/2114) (@joe-elliott)
* [CHANGE] Pad leading zeroes in span id to always be 16 chars [#2062](https://github.com/grafana/tempo/pull/2062) (@ie-pham)

## v2.0.0 / 2023-01-31

* [CHANGE] **BREAKING CHANGE** Use snake case on Azure Storage config [#1879](https://github.com/grafana/tempo/issues/1879) (@faustodavid)
Example of using snake case on Azure Storage config:
```
# config.yaml
storage:
  azure:
    storage_account_name:
    storage_account_key:
    container_name:
```
* [CHANGE] Increase default values for `server.grpc_server_max_recv_msg_size` and `server.grpc_server_max_send_msg_size` from 4MB to 16MB [#1688](https://github.com/grafana/tempo/pull/1688) (@mapno)
* [CHANGE] Propagate Ingesters search errors correctly [#2023](https://github.com/grafana/tempo/pull/2023) (@electron0zero)
* [CHANGE] **BREAKING CHANGE** Use storage.trace.block.row_group_size_bytes to cut rows during compaction instead of
  compactor.compaction.flush_size_bytes. [#1696](https://github.com/grafana/tempo/pull/1696) (@joe-elliott)
* [CHANGE] Update Go to 1.19 [#1665](https://github.com/grafana/tempo/pull/1665) (@ie-pham)
* [CHANGE] Remove unsued scheduler frontend code [#1734](https://github.com/grafana/tempo/pull/1734) (@mapno)
* [CHANGE] Deprecated `query-frontend.query_shards` in favor of `query_frontend.trace_by_id.query_shards`.
Old config will still work but will be removed in a future release. [#1735](https://github.com/grafana/tempo/pull/1735) (@mapno)
* [CHANGE] Update alpine image version to 3.16. [#1784](https://github.com/grafana/tempo/pull/1784) (@zalegrala)
* [CHANGE] Delete TempoRequestErrors alert from mixin [#1810](https://github.com/grafana/tempo/pull/1810) (@zalegrala)
  - **BREAKING CHANGE** Any jsonnet users relying on this alert should copy this into their own environment.
* [CHANGE] Update and replace a few go modules [#1945](https://github.com/grafana/tempo/pull/1945) (@zalegrala)
  * Replace `github.com/thanos-io/thanos/pkg/discovery/dns` use with `github.com/grafana/dskit/dns`
  * Upgrade `github.com/grafana/dskit`
  * Upgrade `github.com/grafana/e2e`
  * Upgrade `github.com/minio/minio-go/v7`
* [CHANGE] Config updates to prepare for Tempo 2.0. [#1978](https://github.com/grafana/tempo/pull/1978) (@joe-elliott)
  Defaults updated:
  ```
  query_frontend:
    max_oustanding_per_tenant: 2000
    search:
        concurrent_jobs: 1000
        target_bytes_per_job: 104857600
        max_duration: 168h
        query_ingesters_until: 30m
    trace_by_id:
        query_shards: 50
  querier:
      max_concurrent_queries: 20
      search:
          prefer_self: 10
  ingester:
      concurrent_flushes: 4
      max_block_duration: 30m
      max_block_bytes: 524288000
  storage:
      trace:
          pool:
              max_workers: 400
              queue_depth: 20000
          search:
              read_buffer_count: 32
              read_buffer_size_bytes: 1048576
  ```
  **BREAKING CHANGE** Renamed/removed/moved
  ```
  query_frontend:
    query_shards:                  // removed. use trace_by_id.query_shards
  querier:
      query_timeout:               // removed. use trace_by_id.query_timeout
  compactor:
      compaction:
          chunk_size_bytes:        // renamed to v2_in_buffer_bytes
          flush_size_bytes:        // renamed to v2_out_buffer_bytes
          iterator_buffer_size:    // renamed to v2_prefetch_traces_count
  ingester:
      use_flatbuffer_search:       // removed. automatically set based on block type
  storage:
      wal:
          encoding:                // renamed to v2_encoding
          version:                 // removed and pinned to block.version
      block:
          index_downsample_bytes:  // renamed to v2_index_downsample_bytes
          index_page_size_bytes:   // renamed to v2_index_page_size_bytes
          encoding:                // renamed to v2_encoding
          row_group_size_bytes:    // renamed to parquet_row_group_size_bytes
  ```
* [CHANGE] **BREAKING CHANGE** Remove `search_enabled` and `metrics_generator_enabled`. Both default to true. [#2004](https://github.com/grafana/tempo/pull/2004) (@joe-elliott)
* [CHANGE] Update OTel collector to v0.57.2 [#1757](https://github.com/grafana/tempo/pull/1757) (@mapno)
* [FEATURE] TraceQL support https://grafana.com/docs/tempo/latest/traceql/
* [FEATURE] Parquet backend is GA and default
* [FEATURE] Add generic forwarder and implement otlpgrpc forwarder [#1775](https://github.com/grafana/tempo/pull/1775) (@Blinkuu)
    New config options and example configuration:
```
# config.yaml
distributor:
  forwarders:
    - name: "otel-forwarder"
      backend: "otlpgrpc"
      otlpgrpc:
        endpoints: ['otelcol:4317']
        tls:
          insecure: true

# overrides.yaml
overrides:
  "example-tenant-1":
    forwarders: ['otel-forwarder']
  "example-tenant-2":
    forwarders: ['otel-forwarder']
```
* [ENHANCEMENT] Add support for TraceQL in Parquet WAL and Local Blocks. [#1966](https://github.com/grafana/tempo/pull/1966) (@electron0zero)
* [ENHANCEMENT] Add `/status/usage-stats` endpoint to show usage stats data [#1782](https://github.com/grafana/tempo/pull/1782) (@electron0zero)
* [ENHANCEMENT] Add TLS support to jaeger query plugin. [#1999](https://github.com/grafana/tempo/pull/1999) (@rubenvp8510)
* [ENHANCEMENT] Collect inspectedBytes from SearchMetrics [#1975](https://github.com/grafana/tempo/pull/1975) (@electron0zero)
* [ENHANCEMENT] Add zone awareness replication for ingesters. [#1936](https://github.com/grafana/tempo/pull/1936) (@manohar-koukuntla)
```
# use the following fields in _config field of jsonnet config, to enable zone aware ingester
    multi_zone_ingester_enabled: false,
    multi_zone_ingester_migration_enabled: false,
    multi_zone_ingester_replicas: 0,
    multi_zone_ingester_max_unavailable: 25,
```
* [ENHANCEMENT] Support global and wildcard overrides in generic forwarder feature [#1871](https://github.com/grafana/tempo/pull/1871) (@Blinkuu)
* [ENHANCEMENT] Add new data-type aware searchtagvalues v2 api [#1956](https://github.com/grafana/tempo/pull/1956) (@mdisibio)
* [ENHANCEMENT] Refactor queueManager into generic queue.Queue [#1796](https://github.com/grafana/tempo/pull/1796) (@Blinkuu)
  - **BREAKING CHANGE** Rename `tempo_distributor_forwarder_queue_length` metric to `tempo_distributor_queue_length`. New metric has two custom labels: `name` and  `tenant`.
  - Deprecated `tempo_distributor_forwarder_pushes_total` metric in favor of `tempo_distributor_queue_pushes_total`.
  - Deprecated `tempo_distributor_forwarder_pushes_failures_total` metric in favor of `tempo_distributor_queue_pushes_failures_total`.
* [ENHANCEMENT] Filter namespace by cluster in tempo dashboards variables [#1771](https://github.com/grafana/tempo/pull/1771) (@electron0zero)
* [ENHANCEMENT] Exit early from sharded search requests [#1742](https://github.com/grafana/tempo/pull/1742) (@electron0zero)
* [ENHANCEMENT] Upgrade prometheus/prometheus to `51a44e6657c3` [#1829](https://github.com/grafana/tempo/pull/1829) (@mapno)
* [ENHANCEMENT] Avoid running tempodb pool jobs with a cancelled context [#1852](https://github.com/grafana/tempo/pull/1852) (@zalegrala)
* [ENHANCEMENT] Add config flag to allow for compactor disablement for debug purposes [#1850](https://github.com/grafana/tempo/pull/1850) (@zalegrala)
* [ENHANCEMENT] Identify bloom that could not be retrieved from backend block [#1737](https://github.com/grafana/tempo/pull/1737) (@AlexDHoffer)
* [ENHANCEMENT] tempo: check configuration returns now a list of warnings [#1663](https://github.com/grafana/tempo/pull/1663) (@frzifus)
* [ENHANCEMENT] Make DNS address fully qualified to reduce DNS lookups in Kubernetes [#1687](https://github.com/grafana/tempo/pull/1687) (@electron0zero)
* [ENHANCEMENT] Improve parquet compaction memory profile when dropping spans [#1692](https://github.com/grafana/tempo/pull/1692) (@joe-elliott)
* [ENHANCEMENT] Use Parquet for local block search, tag search and tag value search instead of flatbuffers. A configuration value
  (`ingester.use_flatbuffer_search`) is provided to continue using flatbuffers.
  - **BREAKING CHANGE** Makes Parquet the default encoding.
* [ENHANCEMENT] Return 200 instead of 206 when blocks failed is < tolerate_failed_blocks. [#1725](https://github.com/grafana/tempo/pull/1725) (@joe-elliott)
* [ENHANCEMENT] Add GOMEMLIMIT variable to compactor jsonnet and set the value to equal compactor memory limit. [#1758](https://github.com/grafana/tempo/pull/1758/files) (@ie-pham)
* [ENHANCEMENT] Add capability to configure the used S3 Storage Class [#1697](https://github.com/grafana/tempo/pull/1714) (@amitsetty)
* [ENHANCEMENT] cache: expose username and sentinel_username redis configuration options for ACL-based Redis Auth support [#1708](https://github.com/grafana/tempo/pull/1708) (@jsievenpiper)
* [ENHANCEMENT] metrics-generator: expose span size as a metric [#1662](https://github.com/grafana/tempo/pull/1662) (@ie-pham)
* [ENHANCEMENT] Set Max Idle connections to 100 for Azure, should reduce DNS errors in Azure [#1632](https://github.com/grafana/tempo/pull/1632) (@electron0zero)
* [ENHANCEMENT] Add PodDisruptionBudget to ingesters in jsonnet [#1691](https://github.com/grafana/tempo/pull/1691) (@joe-elliott)
* [ENHANCEMENT] Add cli command an existing file to tempodb's current parquet schema. [#1706](https://github.com/grafana/tempo/pull/1707) (@joe-elliott)
* [ENHANCEMENT] Add query parameter to search API for traceQL queries [#1729](https://github.com/grafana/tempo/pull/1729) (@kvrhdn)
* [ENHANCEMENT] metrics-generator: filter out older spans before metrics are aggregated [#1612](https://github.com/grafana/tempo/pull/1612) (@ie-pham)
* [ENHANCEMENT] Add hedging to trace by ID lookups created by the frontend. [#1735](https://github.com/grafana/tempo/pull/1735) (@mapno)
    New config options and defaults:
```
query_frontend:
  trace_by_id:
    hedge_requests_at: 5s
    hedge_requests_up_to: 3
```
* [ENHANCEMENT] Vulture now has improved distribution of the random traces it searches. [#1763](https://github.com/grafana/tempo/pull/1763) (@rfratto)
* [ENHANCEMENT] Upgrade opentelemetry-proto submodule to v0.18.0 Internal types are updated to use `scope` instead of `instrumentation_library`. 
                This is a breaking change in trace by ID queries if JSON is requested. [#1754](https://github.com/grafana/tempo/pull/1754) (@mapno)
* [ENHANCEMENT] Add TLS support to the vulture [#1874](https://github.com/grafana/tempo/pull/1874) (@zalegrala)
* [ENHANCEMENT] metrics-generator: extract `status_message` field from spans [#1786](https://github.com/grafana/tempo/pull/1786), [#1794](https://github.com/grafana/tempo/pull/1794) (@stoewer)
* [ENHANCEMENT] metrics-generator: handle collisions between user defined and default dimensions [#1794](https://github.com/grafana/tempo/pull/1794) (@stoewer)
  **BREAKING CHANGE** Custom dimensions colliding with intrinsic dimensions will be prefixed with `__`.
* [ENHANCEMENT] metrics-generator: make intrinsic dimensions configurable and disable `status_message` by default [#1960](https://github.com/grafana/tempo/pull/1960) (@stoewer)
* [ENHANCEMENT] distributor: Log span names when `distributor.log_received_spans.include_all_attributes` is on [#1790](https://github.com/grafana/tempo/pull/1790) (@suraciii)
* [ENHANCEMENT] metrics-generator: truncate label names and values exceeding a configurable length [#1897](https://github.com/grafana/tempo/pull/1897) (@kvrhdn)
* [ENHANCEMENT] Add parquet WAL [#1878](https://github.com/grafana/tempo/pull/1878) (@joe-elliott, @mdisibio)
* [ENHANCEMENT] Convert last few Jsonnet alerts with per_cluster_label [#2000](https://github.com/grafana/tempo/pull/2000) (@Whyeasy)
* [ENHANCEMENT] New tenant dashboard [#1901](https://github.com/grafana/tempo/pull/1901) (@mapno)
* [BUGFIX] Stop distributors on Otel receiver fatal error[#1887](https://github.com/grafana/tempo/pull/1887) (@rdooley)
* [BUGFIX] New wal file separator '+' for the NTFS filesystem and backward compatibility with the old separator ':' [#1700](https://github.com/grafana/tempo/pull/1700) (@kilian-kier)
* [BUGFIX] Honor caching and buffering settings when finding traces by id [#1697](https://github.com/grafana/tempo/pull/1697) (@joe-elliott)
* [BUGFIX] Correctly propagate errors from the iterator layer up through the queriers [#1723](https://github.com/grafana/tempo/pull/1723) (@joe-elliott)
* [BUGFIX] Make multitenancy work with HTTP [#1781](https://github.com/grafana/tempo/pull/1781) (@gouthamve)
* [BUGFIX] Fix parquet search bug fix on http.status_code that may cause incorrect results to be returned [#1799](https://github.com/grafana/tempo/pull/1799) (@mdisibio)
* [BUGFIX] Fix failing SearchTagValues endpoint after startup [#1813](https://github.com/grafana/tempo/pull/1813) (@stoewer)
* [BUGFIX] tempo-mixin: tweak dashboards to support metrics without `cluster` label present [#1913](https://github.com/grafana/tempo/pull/1913) (@kvrhdn)
* [BUGFIX] Fix docker-compose examples not running on Apple M1 hardware [#1920](https://github.com/grafana/tempo/pull/1920) (@stoewer)
* [BUGFIX] Fix traceql parsing of most binary operations to not require spacing [#1939](https://github.com/grafana/tempo/pull/1941) (@mdisibio)
* [BUGFIX] Don't persist tenants without blocks in the ingester[#1947](https://github.com/grafana/tempo/pull/1947) (@joe-elliott)
* [BUGFIX] TraceQL: span scope not working with ranges [#1948](https://github.com/grafana/tempo/issues/1948) (@mdisibio)
* [BUGFIX] TraceQL: skip live traces search [#1997](https://github.com/grafana/tempo/pull/1997) (@mapno)
* [BUGFIX] Return more consistent search results by combining partial traces [#2003](https://github.com/grafana/tempo/pull/2003) (@mapno)

## v1.5.0 / 2022-08-17

* [CHANGE] metrics-generator: Changed added metric label `instance` to `__metrics_gen_instance` to reduce collisions with custom dimensions. [#1439](https://github.com/grafana/tempo/pull/1439) (@joe-elliott)
* [CHANGE] Don't enforce `max_bytes_per_tag_values_query` when set to 0. [#1447](https://github.com/grafana/tempo/pull/1447) (@joe-elliott)
* [CHANGE] Add new querier service in deployment jsonnet to serve `/status` endpoint. [#1474](https://github.com/grafana/tempo/pull/1474) (@annanay25)
* [CHANGE] Swapped out Google Cloud Functions serverless docs and build for Google Cloud Run. [#1483](https://github.com/grafana/tempo/pull/1483) (@joe-elliott)
* [CHANGE] **BREAKING CHANGE** Change spanmetrics metric names and labels to match OTel conventions. [#1478](https://github.com/grafana/tempo/pull/1478) (@mapno)
* [FEATURE] Add support for time picker in jaeger query plugin. [#1631](https://github.com/grafana/tempo/pull/1631) (@rubenvp8510)
Old metric names:
```
traces_spanmetrics_duration_seconds_{sum,count,bucket}
```
New metric names:
```
traces_spanmetrics_latency_{sum,count,bucket}
```
Additionally, default label `span_status` is renamed to `status_code`.
* [CHANGE] Update to Go 1.18 [#1504](https://github.com/grafana/tempo/pull/1504) (@annanay25)
* [CHANGE] Change tag/value lookups to return partial results when reaching response size limit instead of failing [#1517](https://github.com/grafana/tempo/pull/1517) (@mdisibio)
* [CHANGE] Change search to be case-sensitive [#1547](https://github.com/grafana/tempo/issues/1547) (@mdisibio)
* [CHANGE] Relax Hedged request defaults for external endpoints. [#1566](https://github.com/grafana/tempo/pull/1566) (@joe-elliott)
  ```
  querier:
    search:
      external_hedge_requests_at: 4s    -> 8s
      external_hedge_requests_up_to: 3  -> 2
  ```
* [CHANGE] **BREAKING CHANGE** Include emptyDir for metrics generator wal storage in jsonnet [#1556](https://github.com/grafana/tempo/pull/1556) (@zalegrala)
Jsonnet users will now need to specify a storage request and limit for the generator wal.
    _config+:: {
      metrics_generator+: {
        ephemeral_storage_request_size: '10Gi',
        ephemeral_storage_limit_size: '11Gi',
      },
    }
* [CHANGE] Two additional latency buckets added to the default settings for generated spanmetrics. Note that this will increase cardinality when using the defaults. [#1593](https://github.com/grafana/tempo/pull/1593) (@fredr)
* [CHANGE] Mark `log_received_traces` as deprecated. New flag is `log_received_spans`.
  Extend distributor spans logger with optional features to include span attributes and a filter by error status. [#1465](https://github.com/grafana/tempo/pull/1465) (@faustodavid)
* [FEATURE] Add parquet block format [#1479](https://github.com/grafana/tempo/pull/1479) [#1531](https://github.com/grafana/tempo/pull/1531) [#1564](https://github.com/grafana/tempo/pull/1564) (@annanay25, @mdisibio)
* [FEATURE] Add anonymous usage reporting, enabled by default. [#1481](https://github.com/grafana/tempo/pull/1481) (@zalegrala)
**BREAKING CHANGE** As part of the usage stats inclusion, the distributor will also require access to the store.  This is required so the distirbutor can know which cluster it should be reporting membership of.
* [FEATURE] Include messaging systems and databases in service graphs. [#1576](https://github.com/grafana/tempo/pull/1576) (@kvrhdn)
* [ENHANCEMENT] Added the ability to have a per tenant max search duration. [#1421](https://github.com/grafana/tempo/pull/1421) (@joe-elliott)
* [ENHANCEMENT] metrics-generator: expose max_active_series as a metric [#1471](https://github.com/grafana/tempo/pull/1471) (@kvrhdn)
* [ENHANCEMENT] Azure Backend: Add support for authentication with Managed Identities. [#1457](https://github.com/grafana/tempo/pull/1457) (@joe-elliott)
* [ENHANCEMENT] Add metric to track feature enablement [#1459](https://github.com/grafana/tempo/pull/1459) (@zalegrala)
* [ENHANCEMENT] Added s3 config option `insecure_skip_verify` [#1470](https://github.com/grafana/tempo/pull/1470) (@zalegrala)
* [ENHANCEMENT] Added polling option to reduce issues in Azure `blocklist_poll_jitter_ms` [#1518](https://github.com/grafana/tempo/pull/1518) (@joe-elliott)
* [ENHANCEMENT] Add a config to query single ingester instance based on trace id hash for Trace By ID API. (1484)[https://github.com/grafana/tempo/pull/1484] (@sagarwala, @bikashmishra100, @ashwinidulams)
* [ENHANCEMENT] Add blocklist metrics for total backend objects and total backend bytes [#1519](https://github.com/grafana/tempo/pull/1519) (@ie-pham)
* [ENHANCEMENT] Adds `tempo_querier_external_endpoint_hedged_roundtrips_total` to count the total hedged requests [#1558](https://github.com/grafana/tempo/pull/1558) (@joe-elliott)
  **BREAKING CHANGE** Removed deprecated metrics `tempodb_(gcs|s3|azure)_request_duration_seconds` in favor of `tempodb_backend_request_duration_seconds`. These metrics
  have been deprecated since v1.1.
* [ENHANCEMENT] Add tags option for s3 backends.  This allows new objects to be written with the configured tags. [#1442](https://github.com/grafana/tempo/pull/1442) (@stevenbrookes)
* [ENHANCEMENT] metrics-generator: support per-tenant processor configuration [#1434](https://github.com/grafana/tempo/pull/1434) (@kvrhdn)
* [ENHANCEMENT] Include rollout dashboard [#1456](https://github.com/grafana/tempo/pull/1456) (@zalegrala)
* [ENHANCEMENT] Add SentinelPassword configuration for Redis [#1463](https://github.com/grafana/tempo/pull/1463) (@zalegrala)
* [BUGFIX] Fix nil pointer panic when the trace by id path errors. [#1441](https://github.com/grafana/tempo/pull/1441) (@joe-elliott)
* [BUGFIX] Update tempo microservices Helm values example which missed the 'enabled' key for thriftHttp. [#1472](https://github.com/grafana/tempo/pull/1472) (@hajowieland)
* [BUGFIX] Fix race condition in forwarder overrides loop. [1468](https://github.com/grafana/tempo/pull/1468) (@mapno)
* [BUGFIX] Fix v2 backend check on span name to be substring [#1538](https://github.com/grafana/tempo/pull/1538) (@mdisibio)
* [BUGFIX] Fix wal check on span name to be substring [#1548](https://github.com/grafana/tempo/pull/1548) (@mdisibio)
* [BUGFIX] Prevent ingester panic "cannot grow buffer" [#1258](https://github.com/grafana/tempo/issues/1258) (@mdisibio)
* [BUGFIX] metrics-generator: do not remove x-scope-orgid header in single tenant modus [#1554](https://github.com/grafana/tempo/pull/1554) (@kvrhdn)
* [BUGFIX] Fixed issue where backend does not support `root.name` and `root.service.name` [#1589](https://github.com/grafana/tempo/pull/1589) (@kvrhdn)
* [BUGFIX] Fixed ingester to continue starting up after block replay error [#1603](https://github.com/grafana/tempo/issues/1603) (@mdisibio)
* [BUGFIX] Fix issue relating to usage stats and GCS returning empty strings as tenantID [#1625](https://github.com/grafana/tempo/pull/1625) (@ie-pham)

## v1.4.1 / 2022-05-05

* [BUGFIX] metrics-generator: don't inject X-Scope-OrgID header for single-tenant setups [1417](https://github.com/grafana/tempo/pull/1417) (@kvrhdn)
* [BUGFIX] compactor: populate `compaction_objects_combined_total` and `tempo_discarded_spans_total{reason="trace_too_large_to_compact"}` metrics again [1420](https://github.com/grafana/tempo/pull/1420) (@mdisibio)
* [BUGFIX] distributor: prevent panics when concurrently calling `shutdown` to forwarder's queueManager [1422](https://github.com/grafana/tempo/pull/1422) (@mapno)

## v1.4.0 / 2022-04-28

* [CHANGE] Vulture now exercises search at any point during the block retention to test full backend search.
  **BREAKING CHANGE** Dropped `tempo-search-retention-duration` parameter.  [#1297](https://github.com/grafana/tempo/pull/1297) (@joe-elliott)
* [CHANGE] Updated storage.trace.pool.queue_depth default from 200->10000. [#1345](https://github.com/grafana/tempo/pull/1345) (@joe-elliott)
* [CHANGE] Update alpine images to 3.15 [#1330](https://github.com/grafana/tempo/pull/1330) (@zalegrala)
* [CHANGE] Updated flags `-storage.trace.azure.storage-account-name` and `-storage.trace.s3.access_key` to no longer to be considered as secrets [#1356](https://github.com/grafana/tempo/pull/1356) (@simonswine)
* [CHANGE] Add warning threshold for TempoIngesterFlushes and adjust critical threshold [#1354](https://github.com/grafana/tempo/pull/1354) (@zalegrala)
* [CHANGE] Include lambda in serverless e2e tests [#1357](https://github.com/grafana/tempo/pull/1357) (@zalegrala)
* [CHANGE] Replace mixin TempoIngesterFlushes metric to only look at retries [#1354](https://github.com/grafana/tempo/pull/1354) (@zalegrala)
* [CHANGE] Update the jsonnet for single-binary to include clustering [#1391](https://github.com/grafana/tempo/pull/1391) (@zalegrala)
  **BREAKING CHANGE** After this change, the port specification has moved under `$._config.tempo` to avoid global port spec.
* [FEATURE]: v2 object encoding added. This encoding adds a start/end timestamp to every record to reduce proto marshalling and increase search speed.
  **BREAKING CHANGE** After this rollout the distributors will use a new API on the ingesters. As such you must rollout all ingesters before rolling the
  distributors. Also, during this period, the ingesters will use considerably more resources and as such should be scaled up (or incoming traffic should be
  heavily throttled). Once all distributors and ingesters have rolled performance will return to normal. Internally we have observed ~1.5x CPU load on the
  ingesters during the rollout. [#1227](https://github.com/grafana/tempo/pull/1227) (@joe-elliott)
* [FEATURE] Added metrics-generator: an optional components to generate metrics from ingested traces [#1282](https://github.com/grafana/tempo/pull/1282) (@mapno, @kvrhdn)
* [FEATURE] Allow the compaction cycle to be configurable with a default of 30 seconds [#1335](https://github.com/grafana/tempo/pull/1335) (@willdot)
* [FEATURE] Add new config options for setting GCS metadata on new objects [](https://github.com/grafana/tempo/pull/1368) (@zalegrala)
* [ENHANCEMENT] Enterprise jsonnet: add config to create tokengen job explicitly [#1256](https://github.com/grafana/tempo/pull/1256) (@kvrhdn)
* [ENHANCEMENT] Add new scaling alerts to the tempo-mixin [#1292](https://github.com/grafana/tempo/pull/1292) (@mapno)
* [ENHANCEMENT] Improve serverless handler error messages [#1305](https://github.com/grafana/tempo/pull/1305) (@joe-elliott)
* [ENHANCEMENT] Added a configuration option `search_prefer_self` to allow the queriers to do some work while also leveraging serverless in search. [#1307](https://github.com/grafana/tempo/pull/1307) (@joe-elliott)
* [ENHANCEMENT] Make trace combination/compaction more efficient [#1291](https://github.com/grafana/tempo/pull/1291) (@mdisibio)
* [ENHANCEMENT] Add Content-Type headers to query-frontend paths [#1306](https://github.com/grafana/tempo/pull/1306) (@wperron)
* [ENHANCEMENT] Partially persist traces that exceed `max_bytes_per_trace` during compaction [#1317](https://github.com/grafana/tempo/pull/1317) (@joe-elliott)
* [ENHANCEMENT] Make search respect per tenant `max_bytes_per_trace` and added `skippedTraces` to returned search metrics. [#1318](https://github.com/grafana/tempo/pull/1318) (@joe-elliott)
* [ENHANCEMENT] Improve serverless consistency by forcing a GC before returning. [#1324](https://github.com/grafana/tempo/pull/1324) (@joe-elliott)
* [ENHANCEMENT] Add forwarding queue from distributor to metrics-generator. [#1331](https://github.com/grafana/tempo/pull/1331) (@mapno)
* [ENHANCEMENT] Add hedging to queries to external endpoints. [#1350](https://github.com/grafana/tempo/pull/1350) (@joe-elliott)
  New config options and defaults:
  ```
  querier:
    search:
      external_hedge_requests_at: 5s
      external_hedge_requests_up_to: 3
  ```
  ** BREAKING CHANGE **
  Querier options related to search have moved under a `search` block:
  ```
  querier:
   search_query_timeout: 30s
   search_external_endpoints: []
   search_prefer_self: 2
  ```
  becomes
  ```
  querier:
    search:
      query_timeout: 30s
      prefer_self: 2
      external_endpoints: []
  ```
* [ENHANCEMENT] Added tenant ID (instance ID) to `trace too large message`. [#1385](https://github.com/grafana/tempo/pull/1385) (@cristiangsp)
* [ENHANCEMENT] Add a startTime and endTime parameter to the Trace by ID Tempo Query API to improve query performance [#1388](https://github.com/grafana/tempo/pull/1388) (@sagarwala, @bikashmishra100, @ashwinidulams)
* [BUGFIX] Correct issue where Azure "Blob Not Found" errors were sometimes not handled correctly [#1390](https://github.com/grafana/tempo/pull/1390) (@joe-elliott)
* [BUGFIX]: Enable compaction and retention in Tanka single-binary [#1352](https://github.com/grafana/tempo/issues/1352)
* [BUGFIX]: Remove unnecessary PersistentVolumeClaim [#1245](https://github.com/grafana/tempo/issues/1245)
* [BUGFIX] Fixed issue when query-frontend doesn't log request details when request is cancelled [#1136](https://github.com/grafana/tempo/issues/1136) (@adityapwr)
* [BUGFIX] Update OTLP port in examples (docker-compose & kubernetes) from legacy ports (55680/55681) to new ports (4317/4318) [#1294](https://github.com/grafana/tempo/pull/1294) (@mapno)
* [BUGFIX] Fixes min/max time on blocks to be based on span times instead of ingestion time. [#1314](https://github.com/grafana/tempo/pull/1314) (@joe-elliott)
  * Includes new configuration option to restrict the amount of slack around now to update the block start/end time. [#1332](https://github.com/grafana/tempo/pull/1332) (@joe-elliott)
    ```
    storage:
      trace:
        wal:
          ingestion_time_range_slack: 2m0s
    ```
  * Includes a new metric to determine how often this range is exceeded: `tempo_warnings_total{reason="outside_ingestion_time_slack"}`
* [BUGFIX] Prevent data race / ingester crash during searching by trace id by using xxhash instance as a local variable. [#1387](https://github.com/grafana/tempo/pull/1387) (@bikashmishra100, @sagarwala, @ashwinidulams)
* [BUGFIX] Fix spurious "failed to mark block compacted during retention" errors [#1372](https://github.com/grafana/tempo/issues/1372) (@mdisibio)
* [BUGFIX] Fix error message "Writer is closed" by resetting compression writer correctly on the error path. [#1379](https://github.com/grafana/tempo/issues/1379) (@annanay25)

## v1.3.2 / 2022-02-23
* [BUGFIX] Fixed an issue where the query-frontend would corrupt start/end time ranges on searches which included the ingesters [#1295] (@joe-elliott)

## v1.3.1 / 2022-02-02
* [BUGFIX] Fixed panic when using etcd as ring's kvstore [#1260](https://github.com/grafana/tempo/pull/1260) (@mapno)

## v1.3.0 / 2022-01-24
* [FEATURE]: Add support for [inline environments](https://tanka.dev/inline-environments). [#1184](https://github.com/grafana/tempo/pull/1184) @irizzant
* [CHANGE] Search: Add new per-tenant limit `max_bytes_per_tag_values_query` to limit the size of tag-values response. [#1068](https://github.com/grafana/tempo/pull/1068) (@annanay25)
* [CHANGE] Reduce MaxSearchBytesPerTrace `ingester.max-search-bytes-per-trace` default to 5KB [#1129](https://github.com/grafana/tempo/pull/1129) @annanay25
* [CHANGE] **BREAKING CHANGE** The OTEL GRPC receiver's default port changed from 55680 to 4317. [#1142](https://github.com/grafana/tempo/pull/1142) (@tete17)
* [CHANGE] Remove deprecated method `Push` from `tempopb.Pusher` [#1173](https://github.com/grafana/tempo/pull/1173) (@kvrhdn)
* [CHANGE] Upgrade cristalhq/hedgedhttp from v0.6.0 to v0.7.0 [#1159](https://github.com/grafana/tempo/pull/1159) (@cristaloleg)
* [CHANGE] Export trace id constant in api package [#1176](https://github.com/grafana/tempo/pull/1176)
* [CHANGE] GRPC `1.33.3` => `1.38.0` broke compatibility with `gogoproto.customtype`. Enforce the use of gogoproto marshalling/unmarshalling for Tempo, Cortex & Jaeger structs. [#1186](https://github.com/grafana/tempo/pull/1186) (@annanay25)
* [CHANGE] **BREAKING CHANGE** Remove deprecated ingester gRPC endpoint and data encoding. The current data encoding was introduced in v1.0.  If running earlier versions, first upgrade to v1.0 through v1.2 and allow time for all blocks to be switched to the "v1" data encoding. [#1215](https://github.com/grafana/tempo/pull/1215) (@mdisibio)
* [FEATURE] Added support for full backend search. [#1174](https://github.com/grafana/tempo/pull/1174) (@joe-elliott)
  **BREAKING CHANGE** Moved `querier.search_max_result_limit` and `querier.search_default_result_limit` to `query_frontend.search.max_result_limit` and `query_frontend.search.default_result_limit`
* [ENHANCEMENT]: Improve variables expansion support [#1212](https://github.com/grafana/tempo/pull/1212) @irizzant
* [ENHANCEMENT] Expose `upto` parameter on hedged requests for each backend with `hedge_requests_up_to`. [#1085](https://github.com/grafana/tempo/pull/1085) (@joe-elliott)
* [ENHANCEMENT] Search: drop use of TagCache, extract tags and tag values on-demand [#1068](https://github.com/grafana/tempo/pull/1068) (@kvrhdn)
* [ENHANCEMENT] Jsonnet: add `$._config.namespace` to filter by namespace in cortex metrics [#1098](https://github.com/grafana/tempo/pull/1098) (@mapno)
* [ENHANCEMENT] Add middleware to compress frontend HTTP responses with gzip if requested [#1080](https://github.com/grafana/tempo/pull/1080) (@kvrhdn, @zalegrala)
* [ENHANCEMENT] Allow query disablement in vulture [#1117](https://github.com/grafana/tempo/pull/1117) (@zalegrala)
* [ENHANCEMENT] Improve memory efficiency of compaction and block cutting. [#1121](https://github.com/grafana/tempo/pull/1121) [#1130](https://github.com/grafana/tempo/pull/1130) (@joe-elliott)
* [ENHANCEMENT] Include metrics for configured limit overrides and defaults: tempo_limits_overrides, tempo_limits_defaults [#1089](https://github.com/grafana/tempo/pull/1089) (@zalegrala)
* [ENHANCEMENT] Add Envoy Proxy panel to `Tempo / Writes` dashboard [#1137](https://github.com/grafana/tempo/pull/1137) (@kvrhdn)
* [ENHANCEMENT] Reduce compactionCycle to improve performance in large multitenant environments [#1145](https://github.com/grafana/tempo/pull/1145) (@joe-elliott)
* [ENHANCEMENT] Added max_time_per_tenant to allow for independently configuring polling and compaction cycle. [#1145](https://github.com/grafana/tempo/pull/1145) (@joe-elliott)
* [ENHANCEMENT] Add `tempodb_compaction_outstanding_blocks` metric to measure compaction load [#1144](https://github.com/grafana/tempo/pull/1144) (@mapno)
* [ENHANCEMENT] Update mixin to use new backend metric [#1151](https://github.com/grafana/tempo/pull/1151) (@zalegrala)
* [ENHANCEMENT] Make `TempoIngesterFlushesFailing` alert more actionable [#1157](https://github.com/grafana/tempo/pull/1157) (@dannykopping)
* [ENHANCEMENT] Switch open-telemetry/opentelemetry-collector to grafana/opentelemetry-collectorl fork, update it to 0.40.0 and add missing dependencies due to the change [#1142](https://github.com/grafana/tempo/pull/1142) (@tete17)
* [ENHANCEMENT] Allow environment variables for Azure storage credentials [#1147](https://github.com/grafana/tempo/pull/1147) (@zalegrala)
* [ENHANCEMENT] jsonnet: set rollingUpdate.maxSurge to 3 for distributor, frontend and queriers [#1164](https://github.com/grafana/tempo/pull/1164) (@kvrhdn)
* [ENHANCEMENT] Reduce search data file sizes by optimizing contents [#1165](https://github.com/grafana/tempo/pull/1165) (@mdisibio)
* [ENHANCEMENT] Add `tempo_ingester_live_traces` metric [#1170](https://github.com/grafana/tempo/pull/1170) (@mdisibio)
* [ENHANCEMENT] Update compactor ring to automatically forget unhealthy entries [#1178](https://github.com/grafana/tempo/pull/1178) (@mdisibio)
* [ENHANCEMENT] Added the ability to pass ISO8601 date/times for start/end date to tempo-cli query api search [#1208](https://github.com/grafana/tempo/pull/1208) (@joe-elliott)
* [ENHANCEMENT] Prevent writes to large traces even after flushing to disk [#1199](https://github.com/grafana/tempo/pull/1199) (@mdisibio)
* [BUGFIX] Add process name to vulture traces to work around display issues [#1127](https://github.com/grafana/tempo/pull/1127) (@mdisibio)
* [BUGFIX] Fixed issue where compaction sometimes dropped spans. [#1130](https://github.com/grafana/tempo/pull/1130) (@joe-elliott)
* [BUGFIX] Ensure that the admin client jsonnet has correct S3 bucket property. (@hedss)
* [BUGFIX] Publish tenant index age correctly for tenant index writers. [#1146](https://github.com/grafana/tempo/pull/1146) (@joe-elliott)
* [BUGFIX] Ingester startup panic `slice bounds out of range` [#1195](https://github.com/grafana/tempo/issues/1195) (@mdisibio)
* [BUGFIX] Update goreleaser install method to `go install`. [#](https://github.com/grafana/tempo/) (@mapno)
* [BUGFIX] tempo-mixin: remove TempoDB Access panel from `Tempo / Reads`, metrics don't exist anymore [#1218](https://github.com/grafana/tempo/issues/1218) (@kvrhdn)

## v1.2.1 / 2021-11-15
* [BUGFIX] Fix defaults for MaxBytesPerTrace (ingester.max-bytes-per-trace) and MaxSearchBytesPerTrace (ingester.max-search-bytes-per-trace) [#1109](https://github.com/grafana/tempo/pull/1109) (@bitprocessor)
* [BUGFIX] Ignore empty objects during compaction [#1113](https://github.com/grafana/tempo/pull/1113) (@mdisibio)

## v1.2.0 / 2021-11-05
* [CHANGE] **BREAKING CHANGE** Drop support for v0 and v1 blocks. See [1.1 changelog](https://github.com/grafana/tempo/releases/tag/v1.1.0) for details [#919](https://github.com/grafana/tempo/pull/919) (@joe-elliott)
* [CHANGE] Renamed CLI flag from `--storage.trace.maintenance-cycle` to `--storage.trace.blocklist_poll`. This is a **breaking change**  [#897](https://github.com/grafana/tempo/pull/897) (@mritunjaysharma394)
* [CHANGE] update jsonnet alerts and recording rules to use `job_selectors` and `cluster_selectors` for configurable unique identifier labels [#935](https://github.com/grafana/tempo/pull/935) (@kevinschoonover)
* [CHANGE] Modify generated tag keys in Vulture for easier filtering [#934](https://github.com/grafana/tempo/pull/934) (@zalegrala)
* [CHANGE] **BREAKING CHANGE** Consolidate status information onto /status endpoint [ #952 ](https://github.com/grafana/tempo/pull/952) @zalegrala)
  The following endpoints moved.
  `/runtime_config` moved to `/status/runtime_config`
  `/config` moved to `/status/config`
  `/services` moved to `/status/services`
* [CHANGE] **BREAKING CHANGE** Change ingester metric `ingester_bytes_metric_total` in favor of `ingester_bytes_received_total` [#979](https://github.com/grafana/tempo/pull/979) (@mapno)
* [CHANGE] Add troubleshooting language to config for `server.grpc_server_max_recv_msg_size` and `server.grpc_server_max_send_msg_size` when handling large traces [#1023](https://github.com/grafana/tempo/pull/1023) (@thejosephstevens)
* [CHANGE] Parse search query tags from `tags` query parameter [#1055](https://github.com/grafana/tempo/pull/1055) (@kvrhdn)
* [FEATURE] Add ability to search ingesters for traces [#806](https://github.com/grafana/tempo/pull/806) (@mdisibio)
* [FEATURE] Add runtime config handler  [#936](https://github.com/grafana/tempo/pull/936) (@mapno)
* [FEATURE] Search WAL reload and compression(versioned encoding) support [#1000](https://github.com/grafana/tempo/pull/1000) (@annanay25, @mdisibio)
* [FEATURE] Added ability to add a middleware to the OTel receivers' consume function [#1015](http://github.com/grafan/tempo/pull/1015) (@chaudum)
* [FEATURE] Add ScalableSingleBinary operational run mode [#1004](https://github.com/grafana/tempo/pull/1004) (@zalegrala)
* [FEATURE] Added a [jsonnet](https://jsonnet.org) library for Grafana Enterprise Traces (GET) deployments [#1096](https://github.com/grafana/tempo/pull/1096)
* [ENHANCEMENT] Added "query blocks" cli option. [#876](https://github.com/grafana/tempo/pull/876) (@joe-elliott)
* [ENHANCEMENT] Added "search blocks" cli option. [#972](https://github.com/grafana/tempo/pull/972) (@joe-elliott)
* [ENHANCEMENT] Added traceid to `trace too large message`. [#888](https://github.com/grafana/tempo/pull/888) (@mritunjaysharma394)
* [ENHANCEMENT] Add support to tempo workloads to `overrides` from single configmap in microservice mode. [#896](https://github.com/grafana/tempo/pull/896) (@kavirajk)
* [ENHANCEMENT] Make `overrides_config` block name consistent with Loki and Cortex in microservice mode. [#906](https://github.com/grafana/tempo/pull/906) (@kavirajk)
* [ENHANCEMENT] Changes the metrics name from `cortex_runtime_config_last_reload_successful` to `tempo_runtime_config_last_reload_successful` [#945](https://github.com/grafana/tempo/pull/945) (@kavirajk)
* [ENHANCEMENT] Updated config defaults to reflect better capture operational knowledge. [#913](https://github.com/grafana/tempo/pull/913) (@joe-elliott)
  ```
  ingester:
    trace_idle_period: 30s => 10s  # reduce ingester memory requirements with little impact on querying
    flush_check_period: 30s => 10s
  query_frontend:
    query_shards: 2 => 20          # will massively improve performance on large installs
  storage:
    trace:
      wal:
        encoding: none => snappy   # snappy has been tested thoroughly and ready for production use
      block:
        bloom_filter_false_positive: .05 => .01          # will increase total bloom filter size but improve query performance
        bloom_filter_shard_size_bytes: 256KiB => 100 KiB # will improve query performance
  compactor:
    compaction:
      chunk_size_bytes: 10 MiB => 5 MiB  # will reduce compactor memory needs
      compaction_window: 4h => 1h        # will allow more compactors to participate in compaction without substantially increasing blocks
  ```
* [ENHANCEMENT] Make s3 backend readError logic more robust [#905](https://github.com/grafana/tempo/pull/905) (@wei840222)
* [ENHANCEMENT] Include additional detail when searching for traces [#916](https://github.com/grafana/tempo/pull/916) (@zalegrala)
* [ENHANCEMENT] Add `gen index` and `gen bloom` commands to tempo-cli. [#903](https://github.com/grafana/tempo/pull/903) (@annanay25)
* [ENHANCEMENT] Implement trace comparison in Vulture [#904](https://github.com/grafana/tempo/pull/904) (@zalegrala)
* [ENHANCEMENT] Improve zstd read throughput using zstd.Decoder [#948](https://github.com/grafana/tempo/pull/948) (@joe-elliott)
* [ENHANCEMENT] Dedupe search records while replaying WAL [#940](https://github.com/grafana/tempo/pull/940) (@annanay25)
* [ENHANCEMENT] Add status endpoint to list the available endpoints [#938](https://github.com/grafana/tempo/pull/938) (@zalegrala)
* [ENHANCEMENT] Compression updates: Added s2, improved snappy performance [#961](https://github.com/grafana/tempo/pull/961) (@joe-elliott)
* [ENHANCEMENT] Add search block headers [#943](https://github.com/grafana/tempo/pull/943) (@mdisibio)
* [ENHANCEMENT] Add search block headers for wal blocks [#963](https://github.com/grafana/tempo/pull/963) (@mdisibio)
* [ENHANCEMENT] Add support for vulture sending long running traces [#951](https://github.com/grafana/tempo/pull/951) (@zalegrala)
* [ENHANCEMENT] Support global denylist and per-tenant allowlist of tags for search data. [#960](https://github.com/grafana/tempo/pull/960) (@annanay25)
* [ENHANCEMENT] Add `search_query_timeout` to querier config. [#984](https://github.com/grafana/tempo/pull/984) (@kvrhdn)
* [ENHANCEMENT] Include simple e2e test to test searching [#978](https://github.com/grafana/tempo/pull/978) (@zalegrala)
* [ENHANCEMENT] Jsonnet: add `$._config.memcached.memory_limit_mb` [#987](https://github.com/grafana/tempo/pull/987) (@kvrhdn)
* [ENHANCEMENT] Upgrade jsonnet-libs to 1.19 and update tk examples [#1001](https://github.com/grafana/tempo/pull/1001) (@mapno)
* [ENHANCEMENT] Shard tenant index creation by tenant and add functionality to handle stale indexes. [#1005](https://github.com/grafana/tempo/pull/1005) (@joe-elliott)
* [ENHANCEMENT] **BREAKING CHANGE** Support partial results from failed block queries [#1007](https://github.com/grafana/tempo/pull/1007) (@mapno)
  Querier [`GET /querier/api/traces/<traceid>`](https://grafana.com/docs/tempo/latest/api_docs/#query) response's body has been modified
  to return `tempopb.TraceByIDResponse` instead of simply `tempopb.Trace`. This will cause a disruption of the read path during rollout of the change.
* [ENHANCEMENT] Add `search_default_limit` and `search_max_result_limit` to querier config. [#1022](https://github.com/grafana/tempo/pull/1022) [#1044](https://github.com/grafana/tempo/pull/1044) (@kvrhdn)
* [ENHANCEMENT] Add new metric `tempo_distributor_push_duration_seconds` [#1027](https://github.com/grafana/tempo/pull/1027) (@zalegrala)
* [ENHANCEMENT] Add query parameter to show the default config values and the difference between the current values and the defaults. [#1045](https://github.com/grafana/tempo/pull/1045) (@MichelHollands)
* [ENHANCEMENT] Adding metrics around ingester flush retries [#1049](https://github.com/grafana/tempo/pull/1049) (@dannykopping)
* [ENHANCEMENT] Performance: More efficient distributor batching [#1075](https://github.com/grafana/tempo/pull/1075) (@joe-elliott)
* [ENHANCEMENT] Allow search disablement in vulture [#1069](https://github.com/grafana/tempo/pull/1069) (@zalegrala)
* [ENHANCEMENT] Jsonnet: add `$._config.search_enabled`, correctly set `http_api_prefix` in config [#1072](https://github.com/grafana/tempo/pull/1072) (@kvrhdn)
* [ENHANCEMENT] Performance: Remove WAL contention between ingest and searches [#1076](https://github.com/grafana/tempo/pull/1076) (@mdisibio)
* [ENHANCEMENT] Include tempo-cli in the release [#1086](https://github.com/grafana/tempo/pull/1086) (@zalegrala)
* [ENHANCEMENT] Add search on span status [#1093](https://github.com/grafana/tempo/pull/1093) (@mdisibio)
* [ENHANCEMENT] Slightly improved compression performance [#1094](https://github.com/grafana/tempo/pull/1094) (@bboreham)
* [BUGFIX] Update port spec for GCS docker-compose example [#869](https://github.com/grafana/tempo/pull/869) (@zalegrala)
* [BUGFIX] Fix "magic number" errors and other block mishandling when an ingester forcefully shuts down [#937](https://github.com/grafana/tempo/issues/937) (@mdisibio)
* [BUGFIX] Fix compactor memory leak [#806](https://github.com/grafana/tempo/pull/806) (@mdisibio)
* [BUGFIX] Fix an issue with WAL replay of zero-length search data when search is disabled. [#968](https://github.com/grafana/tempo/pull/968) (@annanay25)
* [BUGFIX] Set span's tag `span.kind` to `client` in query-frontend [#975](https://github.com/grafana/tempo/pull/975) (@mapno)
* [BUGFIX] Nil check overrides module in the `/status` handler [#994](https://github.com/grafana/tempo/pull/994) (@mapno)
* [BUGFIX] Several bug fixes for search contention and panics [#1033](https://github.com/grafana/tempo/pull/1033) (@mdisibio)
* [BUGFIX] Fixes `tempodb_backend_hedged_roundtrips_total` to correctly count hedged roundtrips. [#1079](https://github.com/grafana/tempo/pull/1079) (@joe-elliott)
* [BUGFIX] Update go-kit logger package to remove spurious debug logs [#1094](https://github.com/grafana/tempo/pull/1094) (@bboreham)

## v1.1.0 / 2021-08-26
* [CHANGE] Upgrade Cortex from v1.9.0 to v1.9.0-131-ga4bf10354 [#841](https://github.com/grafana/tempo/pull/841) (@aknuds1)
* [CHANGE] Change default tempo port from 3100 to 3200 [#770](https://github.com/grafana/tempo/pull/809) (@MurzNN)
* [CHANGE] Jsonnet: use dedicated configmaps for distributors and ingesters [#775](https://github.com/grafana/tempo/pull/775) (@kvrhdn)
* [CHANGE] Docker images are now prefixed by their branch name [#828](https://github.com/grafana/tempo/pull/828) (@jvrplmlmn)
* [CHANGE] Update to Go 1.17 [#953](https://github.com/grafana/tempo/pull/953)
* [FEATURE] Added the ability to hedge requests with all backends [#750](https://github.com/grafana/tempo/pull/750) (@joe-elliott)
* [FEATURE] Added a tenant index to reduce bucket polling. [#834](https://github.com/grafana/tempo/pull/834) (@joe-elliott)
* [ENHANCEMENT] Added hedged request metric `tempodb_backend_hedged_roundtrips_total` and a new storage agnostic `tempodb_backend_request_duration_seconds` metric that
  supersedes the soon-to-be deprecated storage specific metrics (`tempodb_azure_request_duration_seconds`, `tempodb_s3_request_duration_seconds` and `tempodb_gcs_request_duration_seconds`). [#790](https://github.com/grafana/tempo/pull/790) (@JosephWoodward)
* [ENHANCEMENT] Performance: improve compaction speed with concurrent reads and writes [#754](https://github.com/grafana/tempo/pull/754) (@mdisibio)
* [ENHANCEMENT] Improve readability of cpu and memory metrics on operational dashboard [#764](https://github.com/grafana/tempo/pull/764) (@bboreham)
* [ENHANCEMENT] Add `azure_request_duration_seconds` metric. [#767](https://github.com/grafana/tempo/pull/767) (@JosephWoodward)
* [ENHANCEMENT] Add `s3_request_duration_seconds` metric. [#776](https://github.com/grafana/tempo/pull/776) (@JosephWoodward)
* [ENHANCEMENT] Add `tempo_ingester_flush_size_bytes` metric. [#777](https://github.com/grafana/tempo/pull/777) (@bboreham)
* [ENHANCEMENT] Microservices jsonnet: resource requests and limits can be set in `$._config`. [#793](https://github.com/grafana/tempo/pull/793) (@kvrhdn)
* [ENHANCEMENT] Add `-config.expand-env` cli flag to support environment variables expansion in config file. [#796](https://github.com/grafana/tempo/pull/796) (@Ashmita152)
* [ENHANCEMENT] Add ability to control bloom filter caching based on age and/or compaction level. Add new cli command `list cache-summary`. [#805](https://github.com/grafana/tempo/pull/805) (@annanay25)
* [ENHANCEMENT] Emit traces for ingester flush operations. [#812](https://github.com/grafana/tempo/pull/812) (@bboreham)
* [ENHANCEMENT] Add retry middleware in query-frontend. [#814](https://github.com/grafana/tempo/pull/814) (@kvrhdn)
* [ENHANCEMENT] Add `-use-otel-tracer` to use the OpenTelemetry tracer, this will also capture traces emitted by the gcs sdk. Experimental: not all features are supported (i.e. remote sampling). [#842](https://github.com/grafana/tempo/pull/842) (@kvrhdn)
* [ENHANCEMENT] Add `/services` endpoint. [#863](https://github.com/grafana/tempo/pull/863) (@kvrhdn)
* [ENHANCEMENT] Include distributed docker-compose example [#859](https://github.com/grafana/tempo/pull/859) (@zalegrala)
* [ENHANCEMENT] Added "query blocks" cli option. [#876](https://github.com/grafana/tempo/pull/876) (@joe-elliott)
* [ENHANCEMENT] Add e2e integration test for GCS. [#883](https://github.com/grafana/tempo/pull/883) (@annanay25)
* [ENHANCEMENT] Added traceid to `trace too large message`. [#888](https://github.com/grafana/tempo/pull/888) (@mritunjaysharma394)
* [ENHANCEMENT] Add support to tempo workloads to `overrides` from single configmap in microservice mode. [#896](https://github.com/grafana/tempo/pull/896) (@kavirajk)
* [ENHANCEMENT] Make `overrides_config` block name consistent with Loki and Cortex in microservice mode. [#906](https://github.com/grafana/tempo/pull/906) (@kavirajk)
* [ENHANCEMENT] Make `overrides_config` mount name static `tempo-overrides` in the tempo workloads in microservice mode. [#906](https://github.com/grafana/tempo/pull/914) (@kavirajk)
* [ENHANCEMENT] Reduce compactor memory usage by forcing garbage collection. [#915](https://github.com/grafana/tempo/pull/915) (@joe-elliott)
* [ENHANCEMENT] Implement search in vulture. [#944](https://github.com/grafana/tempo/pull/944) (@zalegrala)
* [BUGFIX] Allow only valid trace ID characters when decoding [#854](https://github.com/grafana/tempo/pull/854) (@zalegrala)
* [BUGFIX] Queriers complete one polling cycle before finishing startup. [#834](https://github.com/grafana/tempo/pull/834) (@joe-elliott)
* [BUGFIX] Update port spec for GCS docker-compose example [#869](https://github.com/grafana/tempo/pull/869) (@zalegrala)
* [BUGFIX] Cortex upgrade to fix an issue where unhealthy compactors can't be forgotten [#878](https://github.com/grafana/tempo/pull/878) (@joe-elliott)


## v1.0.1 / 2021-06-14

* [BUGFIX] Guard against negative dataLength [#763](https://github.com/grafana/tempo/pull/763) (@joe-elliott)

## v1.0.0 / 2021-06-08

* [CHANGE] Mark `-auth.enabled` as deprecated. New flag is `-multitenancy.enabled` and is set to false by default.
  This is a **breaking change** if you were relying on auth/multitenancy being enabled by default. [#646](https://github.com/grafana/tempo/pull/646)
* [ENHANCEMENT] Performance: Improve Ingester Record Insertion. [#681](https://github.com/grafana/tempo/pull/681)
* [ENHANCEMENT] Improve WAL Replay by not rebuilding the WAL. [#668](https://github.com/grafana/tempo/pull/668)
* [ENHANCEMENT] Add config option to disable write extension to the ingesters. [#677](https://github.com/grafana/tempo/pull/677)
* [ENHANCEMENT] Preallocate byte slices on ingester request unmarshal. [#679](https://github.com/grafana/tempo/pull/679)
* [ENHANCEMENT] Reduce marshalling in the ingesters to improve performance. [#694](https://github.com/grafana/tempo/pull/694)
  This change requires a specific rollout process to prevent dropped spans. First, rollout everything except distributors. After all ingesters have updated
  you can then rollout distributors to the latest version. This is due to changes in the communication between ingesters <-> distributors.
* [ENHANCEMENT] Allow setting the bloom filter shard size with support dynamic shard count.[#644](https://github.com/grafana/tempo/pull/644)
* [ENHANCEMENT] GCS SDK update v1.12.0 => v.15.0, ReadAllWithEstimate used in GCS/S3 backends. [#693](https://github.com/grafana/tempo/pull/693)
* [ENHANCEMENT] Add a new endpoint `/api/echo` to test the query frontend is reachable. [#714](https://github.com/grafana/tempo/pull/714)
* [BUGFIX] Fix Query Frontend grpc settings to avoid noisy error log. [#690](https://github.com/grafana/tempo/pull/690)
* [BUGFIX] Zipkin Support - CombineTraces. [#688](https://github.com/grafana/tempo/pull/688)
* [BUGFIX] Zipkin support - Dedupe span IDs based on span.Kind (client/server) in Query Frontend. [#687](https://github.com/grafana/tempo/pull/687)
* [BUGFIX] Azure Backend - Fix an issue with the append method on the Azure backend. [#736](https://github.com/grafana/tempo/pull/736)

## v0.7.0 / 2021-04-22

**License Change** v0.7.0 and future versions are licensed under AGPLv3 [#660](https://github.com/grafana/tempo/pull/660)

* [CHANGE] Add `json` struct tags to overrides' `Limits` struct in addition to `yaml` tags. [#656](https://github.com/grafana/tempo/pull/656)
* [CHANGE] Update to Go 1.16, latest OpenTelemetry proto definition and collector [#546](https://github.com/grafana/tempo/pull/546)
* [CHANGE] `max_spans_per_trace` limit override has been removed in favour of `max_bytes_per_trace`.
  This is a **breaking change** to the overrides config section. [#612](https://github.com/grafana/tempo/pull/612)
* [CHANGE] Add new flag `-ingester.lifecycler.ID` to manually override the ingester ID with which to register in the ring. [#625](https://github.com/grafana/tempo/pull/625)
* [CHANGE] `ingestion_rate_limit` limit override has been removed in favour of `ingestion_rate_limit_bytes`.
  `ingestion_burst_size` limit override has been removed in favour of `ingestion_burst_size_bytes`.
  This is a **breaking change** to the overrides config section. [#630](https://github.com/grafana/tempo/pull/630)
* [FEATURE] Add page based access to the index file. [#557](https://github.com/grafana/tempo/pull/557)
* [FEATURE] (Experimental) WAL Compression/checksums. [#638](https://github.com/grafana/tempo/pull/638)
* [ENHANCEMENT] Add a Shutdown handler to flush data to backend, at "/shutdown". [#526](https://github.com/grafana/tempo/pull/526)
* [ENHANCEMENT] Queriers now query all (healthy) ingesters for a trace to mitigate 404s on ingester rollouts/scaleups.
  This is a **breaking change** and will likely result in query errors on rollout as the query signature b/n QueryFrontend & Querier has changed. [#557](https://github.com/grafana/tempo/pull/557)
* [ENHANCEMENT] Add list compaction-summary command to tempo-cli [#588](https://github.com/grafana/tempo/pull/588)
* [ENHANCEMENT] Add list and view index commands to tempo-cli [#611](https://github.com/grafana/tempo/pull/611)
* [ENHANCEMENT] Add a configurable prefix for HTTP endpoints. [#631](https://github.com/grafana/tempo/pull/631)
* [ENHANCEMENT] Add kafka receiver. [#613](https://github.com/grafana/tempo/pull/613)
* [ENHANCEMENT] Upgrade OTel collector to `v0.21.0`. [#613](https://github.com/grafana/tempo/pull/627)
* [ENHANCEMENT] Add support for Cortex Background Cache. [#640](https://github.com/grafana/tempo/pull/640)
* [BUGFIX] Fixes permissions errors on startup in GCS. [#554](https://github.com/grafana/tempo/pull/554)
* [BUGFIX] Fixes error where Dell ECS cannot list objects. [#561](https://github.com/grafana/tempo/pull/561)
* [BUGFIX] Fixes listing blocks in S3 when the list is truncated. [#567](https://github.com/grafana/tempo/pull/567)
* [BUGFIX] Fixes where ingester may leave file open [#570](https://github.com/grafana/tempo/pull/570)
* [BUGFIX] Fixes a bug where some blocks were not searched due to query sharding and randomness in blocklist poll. [#583](https://github.com/grafana/tempo/pull/583)
* [BUGFIX] Fixes issue where wal was deleted before successful flush and adds exponential backoff for flush errors [#593](https://github.com/grafana/tempo/pull/593)
* [BUGFIX] Fixes issue where Tempo would not parse odd length trace ids [#605](https://github.com/grafana/tempo/pull/605)
* [BUGFIX] Sort traces on flush to reduce unexpected recombination work by compactors [#606](https://github.com/grafana/tempo/pull/606)
* [BUGFIX] Ingester fully persists blocks locally to reduce amount of work done after restart [#628](https://github.com/grafana/tempo/pull/628)

## v0.6.0 / 2021-02-18

* [CHANGE] Fixed ingester latency spikes on read [#461](https://github.com/grafana/tempo/pull/461)
* [CHANGE] Ingester cut blocks based on size instead of trace count.  Replace ingester `traces_per_block` setting with `max_block_bytes`. This is a **breaking change**. [#474](https://github.com/grafana/tempo/issues/474)
* [CHANGE] Refactor cache section in tempodb. This is a **breaking change** b/c the cache config section has changed. [#485](https://github.com/grafana/tempo/pull/485)
* [CHANGE] New compactor setting for max block size data instead of traces. [#520](https://github.com/grafana/tempo/pull/520)
* [CHANGE] Change default ingester_client compression from gzip to snappy. [#522](https://github.com/grafana/tempo/pull/522)
* [CHANGE/BUGFIX] Rename `tempodb_compaction_objects_written` and `tempodb_compaction_bytes_written` metrics to `tempodb_compaction_objects_written_total` and `tempodb_compaction_bytes_written_total`. [#524](https://github.com/grafana/tempo/pull/524)
* [CHANGE] Replace tempo-cli `list block` `--check-dupes` option with `--scan` and collect additional stats [#534](https://github.com/grafana/tempo/pull/534)
* [FEATURE] Added block compression.  This is a **breaking change** b/c some configuration fields moved. [#504](https://github.com/grafana/tempo/pull/504)
* [CHANGE] Drop Vulture Loki dependency. This is a **breaking change**. [#509](https://github.com/grafana/tempo/pull/509)
* [ENHANCEMENT] Serve config at the "/config" endpoint. [#446](https://github.com/grafana/tempo/pull/446)
* [ENHANCEMENT] Switch blocklist polling and retention to different concurrency mechanism, add configuration options. [#475](https://github.com/grafana/tempo/issues/475)
* [ENHANCEMENT] Add S3 options region and forcepathstyle [#431](https://github.com/grafana/tempo/issues/431)
* [ENHANCEMENT] Add exhaustive search to combine traces from all blocks in the backend. [#489](https://github.com/grafana/tempo/pull/489)
* [ENHANCEMENT] Add per-tenant block retention [#77](https://github.com/grafana/tempo/issues/77)
* [ENHANCEMENT] Change index-downsample to index-downsample-bytes.  This is a **breaking change** [#519](https://github.com/grafana/tempo/issues/519)
* [BUGFIX] Upgrade cortex dependency to v1.7.0-rc.0+ to address issue with forgetting ring membership [#442](https://github.com/grafana/tempo/pull/442) [#512](https://github.com/grafana/tempo/pull/512)
* [BUGFIX] No longer raise the `tempodb_blocklist_poll_errors_total` metric if a block doesn't have meta or compacted meta. [#481](https://github.com/grafana/tempo/pull/481)]
* [BUGFIX] Replay wal completely before ingesting new spans. [#525](https://github.com/grafana/tempo/pull/525)

## v0.5.0 / 2021-01-15

* [CHANGE] Redo tempo-cli with basic command structure and improvements [#385](https://github.com/grafana/tempo/pull/385)
* [CHANGE] Add content negotiation support and sharding parameters to Querier [#375](https://github.com/grafana/tempo/pull/375)
* [CHANGE] Remove S3 automatic bucket creation [#404](https://github.com/grafana/tempo/pull/404)
* [CHANGE] Compactors should round robin tenants instead of choosing randomly [#420](https://github.com/grafana/tempo/issues/420)
* [CHANGE] Switch distributor->ingester communication to more efficient PushBytes method.  This is a **breaking change** when running in microservices mode with separate distributors and ingesters.  To prevent errors ingesters must be fully upgraded first, then distributors.
* [CHANGE] Removed disk_cache.  This is a **breaking change** b/c there is no disk cache. Please use redis or memcached. [#441](https://github.com/grafana/tempo/pull/441)
* [CHANGE] Rename IngestionMaxBatchSize to IngestionBurstSize. This is a **breaking change**. [#445](https://github.com/grafana/tempo/pull/445)
* [ENHANCEMENT] Add docker-compose example for GCS along with new backend options [#397](https://github.com/grafana/tempo/pull/397)
* [ENHANCEMENT] tempo-cli list blocks usability improvements [#403](https://github.com/grafana/tempo/pull/403)
* [ENHANCEMENT] Reduce active traces locking time. [#449](https://github.com/grafana/tempo/pull/449)
* [ENHANCEMENT] Added `tempo_distributor_bytes_received_total` as a per tenant counter of uncompressed bytes received. [#453](https://github.com/grafana/tempo/pull/453)
* [BUGFIX] Compactor without GCS permissions fail silently [#379](https://github.com/grafana/tempo/issues/379)
* [BUGFIX] Prevent race conditions between querier polling and ingesters clearing complete blocks [#421](https://github.com/grafana/tempo/issues/421)
* [BUGFIX] Exclude blocks in last active window from compaction [#411](https://github.com/grafana/tempo/pull/411)
* [BUGFIX] Mixin: Ignore metrics and query-frontend route when checking for TempoRequestLatency alert. [#440](https://github.com/grafana/tempo/pull/440)
* [FEATURE] Add support for Azure Blob Storage backend [#340](https://github.com/grafana/tempo/issues/340)
* [FEATURE] Add Query Frontend module to allow scaling the query path [#400](https://github.com/grafana/tempo/pull/400)

## v0.4.0 / 2020-12-03

* [CHANGE] From path.Join to filepath.Join [#338](https://github.com/grafana/tempo/pull/338)
* [CHANGE] Upgrade Cortex from v1.3.0 to v.1.4.0 [#341](https://github.com/grafana/tempo/pull/341)
* [CHANGE] Compact more than 2 blocks at a time [#348](https://github.com/grafana/tempo/pull/348)
* [CHANGE] Remove tempodb_compaction_duration_seconds metric. [#360](https://github.com/grafana/tempo/pull/360)
* [ENHANCEMENT] Add tempodb_compaction_objects_combined metric. [#339](https://github.com/grafana/tempo/pull/339)
* [ENHANCEMENT] Added OpenMetrics exemplar support. [#359](https://github.com/grafana/tempo/pull/359)
* [ENHANCEMENT] Add tempodb_compaction_objects_written metric. [#360](https://github.com/grafana/tempo/pull/360)
* [ENHANCEMENT] Add tempodb_compaction_bytes_written metric. [#360](https://github.com/grafana/tempo/pull/360)
* [ENHANCEMENT] Add tempodb_compaction_blocks_total metric. [#360](https://github.com/grafana/tempo/pull/360)
* [ENHANCEMENT] Add support for S3 V2 signatures. [#352](https://github.com/grafana/tempo/pull/352)
* [ENHANCEMENT] Add support for Redis caching. [#354](https://github.com/grafana/tempo/pull/354)
* [BUGFIX] Frequent errors logged by compactor regarding meta not found [#327](https://github.com/grafana/tempo/pull/327)
* [BUGFIX] Fix distributors panicking on rollout [#343](https://github.com/grafana/tempo/pull/343)
* [BUGFIX] Fix ingesters occassionally double flushing [#364](https://github.com/grafana/tempo/pull/364)
* [BUGFIX] Fix S3 backend logs "unsupported value type" [#381](https://github.com/grafana/tempo/issues/381)

## v0.3.0 / 2020-11-10

* [CHANGE] Bloom filters are now sharded to reduce size and improve caching, as blocks grow. This is a **breaking change** and all data stored before this change will **not** be queryable. [#192](https://github.com/grafana/tempo/pull/192)
* [CHANGE] Rename maintenance cycle to blocklist poll. [#315](https://github.com/grafana/tempo/pull/315)
* [ENHANCEMENT] CI checks for vendored dependencies using `make vendor-check`. Update CONTRIBUTING.md to reflect the same before checking in files in a PR. [#274](https://github.com/grafana/tempo/pull/274)
* [ENHANCEMENT] Add warnings for suspect configs. [#294](https://github.com/grafana/tempo/pull/294)
* [ENHANCEMENT] Add command line flags for s3 credentials. [#308](https://github.com/grafana/tempo/pull/308)
* [ENHANCEMENT] Support multiple authentication methods for S3 (IRSA, IAM role, static). [#320](https://github.com/grafana/tempo/pull/320)
* [ENHANCEMENT] Add  per tenant bytes counter. [#331](https://github.com/grafana/tempo/pull/331)
* [BUGFIX] S3 multi-part upload errors [#306](https://github.com/grafana/tempo/pull/325)
* [BUGFIX] Increase Prometheus `notfound` metric on tempo-vulture. [#301](https://github.com/grafana/tempo/pull/301)
* [BUGFIX] Return 404 if searching for a tenant id that does not exist in the backend. [#321](https://github.com/grafana/tempo/pull/321)
* [BUGFIX] Prune in-memory blocks from missing tenants. [#314](https://github.com/grafana/tempo/pull/314)
