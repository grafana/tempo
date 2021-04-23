## main / unreleased

## v0.7.0

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

## v0.6.0

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

## v0.5.0

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

## v0.4.0

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

## v0.3.0

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
