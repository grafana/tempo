---
title: Compression and encoding
weight: 50
---

<!-- Page needs to be updated. -->

# Compression and encoding

Tempo can compress traces that it pushes to backend storage. This requires extra
memory and CPU, but it reduces the quantity of stored data.  Anecdotal tests suggest that `zstd` will
cut your storage costs to ~15% of the uncompressed amount.  It is _highly_ recommended to use the
default `zstd`.

Compression is configured under storage like so:

```yaml
storage:
  trace:
    block:
      v2_encoding: zstd
```

The following options are supported:

- none
- gzip
- lz4-64k
- lz4-256k
- lz4-1M
- lz4
- snappy
- zstd
- s2

It is important to note that although all of these compression formats are supported in Tempo, at Grafana
we use `zstd` and it's possible/probable that the other compression algorithms may have issue at scale.
File an issue if you have any problems.

## WAL

The WAL also supports compression. By default, this is configured to use `snappy`. This comes with a small performance
penalty but reduces disk I/O and and adds checksums to the WAL. All of the above configuration options are supported
but only `snappy` has been tested at scale.

```
storage:
  trace:
    wal:
      v2_encoding: snappy
```