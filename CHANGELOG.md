## master / unreleased

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
