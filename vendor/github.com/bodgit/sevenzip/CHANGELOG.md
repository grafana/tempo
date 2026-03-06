# Changelog

## [1.6.0](https://github.com/bodgit/sevenzip/compare/v1.5.2...v1.6.0) (2024-11-17)


### Features

* Add ReadError to wrap I/O errors ([#278](https://github.com/bodgit/sevenzip/issues/278)) ([d38d0aa](https://github.com/bodgit/sevenzip/commit/d38d0aaf74e642d9004b8fee09ab93befeffd174))

## [1.5.2](https://github.com/bodgit/sevenzip/compare/v1.5.1...v1.5.2) (2024-08-29)


### Bug Fixes

* Avoid panic in Reader init (empty2.7z); header.filesInfo is nil. ([#252](https://github.com/bodgit/sevenzip/issues/252)) ([10d7550](https://github.com/bodgit/sevenzip/commit/10d75506fa01719e9e0f074c4e7b3c3b96f4233d))
* Lint fixes ([#253](https://github.com/bodgit/sevenzip/issues/253)) ([c82d2e9](https://github.com/bodgit/sevenzip/commit/c82d2e90e52ae81797b0f790fabe90baf35bf581))

## [1.5.1](https://github.com/bodgit/sevenzip/compare/v1.5.0...v1.5.1) (2024-04-05)


### Performance Improvements

* Add AES key caching ([#189](https://github.com/bodgit/sevenzip/issues/189)) ([3d794c2](https://github.com/bodgit/sevenzip/commit/3d794c26c683fe80def4496d49106679b868ae2e))
* Don't use pools for streams with one file ([#194](https://github.com/bodgit/sevenzip/issues/194)) ([b4cfdcf](https://github.com/bodgit/sevenzip/commit/b4cfdcfe0a64380d64c112d41a870dc8c33c1274))

## [1.5.0](https://github.com/bodgit/sevenzip/compare/v1.4.5...v1.5.0) (2024-02-08)


### Features

* Export the folder/stream identifier ([#169](https://github.com/bodgit/sevenzip/issues/169)) ([187a49e](https://github.com/bodgit/sevenzip/commit/187a49e243ec0618b527851fcee0503d8436e7c2))

## [1.4.5](https://github.com/bodgit/sevenzip/compare/v1.4.4...v1.4.5) (2023-12-12)


### Bug Fixes

* Handle lack of CRC digests ([#143](https://github.com/bodgit/sevenzip/issues/143)) ([4ead944](https://github.com/bodgit/sevenzip/commit/4ead944ad71398931b70a09ea40ba9ce742f4bf7))
* Handle small reads in branch converters ([#144](https://github.com/bodgit/sevenzip/issues/144)) ([dfaf538](https://github.com/bodgit/sevenzip/commit/dfaf538402be45e6cd12064b3d49e7496d2b22f4))

## [1.4.4](https://github.com/bodgit/sevenzip/compare/v1.4.3...v1.4.4) (2023-11-06)


### Bug Fixes

* Handle panic when unpack info is missing ([#117](https://github.com/bodgit/sevenzip/issues/117)) ([db3ba77](https://github.com/bodgit/sevenzip/commit/db3ba775286aa4efce8fdd1c398bf2bd4dfba37d))
