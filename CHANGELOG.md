## main / unreleased

* [CHANGE] **BREAKING CHANGE** Drop support for v0 and v1 blocks. See [1.1 changelog](https://github.com/grafana/tempo/releases/tag/v1.1.0) for details [#919](https://github.com/grafana/tempo/pull/919) (@joe-elliott)
* [FEATURE] Add ability to search ingesters for traces [#806](https://github.com/grafana/tempo/pull/806) (@mdisibio)
* [BUGFIX] Update port spec for GCS docker-compose example [#869](https://github.com/grafana/tempo/pull/869) (@zalegrala)
* [BUGFIX] Cortex upgrade to fix an issue where unhealthy compactors can't be forgotten [#878](https://github.com/grafana/tempo/pull/878) (@joe-elliott)
* [BUGFIX] Fix "magic number" errors and other block mishandling when an ingester forcefully shuts down [#937](https://github.com/grafana/tempo/issues/937) (@mdisibio)
* [ENHANCEMENT] Added "query blocks" cli option. [#876](https://github.com/grafana/tempo/pull/876) (@joe-elliott)
* [ENHANCEMENT] Added traceid to `trace too large message`. [#888](https://github.com/grafana/tempo/pull/888) (@mritunjaysharma394)
* [ENHANCEMENT] Add support to tempo workloads to `overrides` from single configmap in microservice mode. [#896](https://github.com/grafana/tempo/pull/896) (@kavirajk)
* [ENHANCEMENT] Make `overrides_config` block name consistent with Loki and Cortex in microservice mode. [#906](https://github.com/grafana/tempo/pull/906) (@kavirajk)
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
* [CHANGE] Renamed CLI flag from `--storage.trace.maintenance-cycle` to `--storage.trace.blocklist_poll`. This is a **breaking change**  [#897](https://github.com/grafana/tempo/pull/897) (@mritunjaysharma394)
* [CHANGE] update jsonnet alerts and recording rules to use `job_selectors` and `cluster_selectors` for configurable unique identifier labels [#935](https://github.com/grafana/tempo/pull/935) (@kevinschoonover)
* [CHANGE] Modify generated tag keys in Vulture for easier filtering [#934](https://github.com/grafana/tempo/pull/934) (@zalegrala)

## v1.1.0 / 2021-08-26
* [CHANGE] Upgrade Cortex from v1.9.0 to v1.9.0-131-ga4bf10354 [#841](https://github.com/grafana/tempo/pull/841) (@aknuds1)
* [CHANGE] Change default tempo port from 3100 to 3200 [#770](https://github.com/grafana/tempo/pull/809) (@MurzNN)
* [CHANGE] Jsonnet: use dedicated configmaps for distributors and ingesters [#775](https://github.com/grafana/tempo/pull/775) (@kvrhdn)
* [CHANGE] Docker images are now prefixed by their branch name [#828](https://github.com/grafana/tempo/pull/828) (@jvrplmlmn)
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
* [ENHANCEMENT] Added traceid to `trace too large message`. [#888](https://github.com/grafana/tempo/pull/888) (@mritunjaysharma394)
* [ENHANCEMENT] Add support to tempo workloads to `overrides` from single configmap in microservice mode. [#896](https://github.com/grafana/tempo/pull/896) (@kavirajk)
* [ENHANCEMENT] Make `overrides_config` block name consistent with Loki and Cortex in microservice mode. [#906](https://github.com/grafana/tempo/pull/906) (@kavirajk)
* [ENHANCEMENT] Make `overrides_config` mount name static `tempo-overrides` in the tempo workloads in microservice mode. [#906](https://github.com/grafana/tempo/pull/914) (@kavirajk)
* [ENHANCEMENT] Reduce compactor memory usage by forcing garbage collection. [#915](https://github.com/grafana/tempo/pull/915) (@joe-elliott)
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
