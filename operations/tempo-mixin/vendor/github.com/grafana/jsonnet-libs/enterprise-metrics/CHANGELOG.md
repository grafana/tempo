# Changelog

All notable changes to this library will be documented in this file.

Entries should be ordered as follows:
- [CHANGE]
- [FEATURE]
- [ENHANCEMENT]
- [BUGFIX]

Entries should include a reference to the Pull Request that introduced the change.

## Unreleased

- [CHANGE] Use of the deprecated `-bootstrap.license.path` flag has been replaced with `-license.path`. #645 
- [CHANGE] The tokengen configuration is now hidden by default to avoid confusing errors when immutable fields are changed. #541
- [CHANGE] Arbitrary storage class names are removed from PersistentVolumeClaims. If you were using those storage class names, you will need to configure the storageClassName in the persistentVolume object. For example in `$.ingester.persistentVolumeClaim`. #577
- [CHANGE] Enabled the self-monitoring feature, which was part of the GEM 1.4 release, by default. #608
- [CHANGE] Migrate to Grafana Mimir Jsonnet lib in place of Cortex #760
  - Enable query scheduler by default
  - Enable query sharding by default
- [FEATURE] Upgrade to [Grafana Enterprise Metrics v1.3.0](https://grafana.com/docs/enterprise-metrics/v1.4.x/downloads/#v130----april-26th-2021). #552
- [FEATURE] Alertmanager and ruler Kubernetes App manifests are now included. #541
- [FEATURE] PersistentVolumeClaims can be configured on all components run as a StatefulSet. This includes the alertmanager, compactor, ingester, and store-gateway components. #577
- [FEATURE] Upgrade to [Grafana Enterprise Metrics v1.4.0](https://grafana.com/docs/enterprise-metrics/v1.4.x/downloads/#v140----june-28th-2021). #598
- [FEATURE] Upgrade to [Grafana Enterprise Metrics v1.4.1](https://grafana.com/docs/enterprise-metrics/v1.4.x/downloads/#v141----june-29th-2021). #608
- [FEATURE] Run the Grafana Enterprise Metrics `overrides-exporter` target as a deployment. #626
- [FEATURE] Upgrade to [Grafana Enterprise Metrics v1.5.0](https://grafana.com/docs/enterprise-metrics/v1.5.x/downloads/#v151----september-21st-2021). #638
- [FEATURE] Upgrade to [Grafana Enterprise Metrics v2.0.0](https://grafana.com/docs/enterprise-metrics/v2.0.x/downloads/#v200----april-13th-2022). #760
  A few notable changes to defaults have been made in GEM 2.0. Consider whether you are currently relying on any of the following defaults before upgrading:
  - Default HTTP port for all components and services has changed to 8080 (from 80)
  - blocks_storage.backend used to be s3, but is now filesystem
  - server.http_listen_port used to be 80 but is now 8080
  - auth.type used to be trust, but is now enterprise
  - ruler_storage.backend used to be s3 but is now filesystem
  - alertmanager_storage.backend used to be s3 but is now filesystem
  - See [GEM 2.0 release notes](https://grafana.com/docs/enterprise-metrics/latest/release-notes/v2-0/) for more information
- [FEATURE] Upgrade to [Grafana Enterprise Metrics v2.0.1](https://grafana.com/docs/enterprise-metrics/v2.0.x/downloads/#v201----april-14th-2022). #767
- [FEATURE] Add memcached for frontend cache #760
- [ENHANCEMENT] Make Secret name for GEM license configurable via `licenseSecretName` #760
- [ENHANCEMENT] Make Secret name for GEM admin token configurable via `adminTokenSecretName`. #600
- [BUGFIX] The `gossip-ring` Service now publishes not ready addresses. #523
- [BUGFIX] All ring members now have the `gossip_ring-member` label set. #523
- [BUGFIX] All microservices now use the `gossip-ring` Service as `memberlist.join` address. #523
- [BUGFIX] Enable admin-api leader election. This change does not affect single replica deployments of the admin-api but does fix the potential for an inconsistent state when running with multiple replicas of the admin-api and experiencing parallel writes for the same objects.
- [BUGFIX] Ensure only a single tokengen Pod is created by having the container restart on failure. #647

## 1.0.0

- [FEATURE] Initial versioned release. #TODO
