---
title: Compression/Encoding
---

Tempo has the ability to compress traces that it pushes into the backend. This requires a bit extra
memory and cpu but seriously reduces the amount of stored data.  Anecdotal tests suggest that zstd will
cut your storage costs to ~15% of the uncompressed amount.  It is _highly_ recommended to use the
default zstd.

Compression is configured under storage like so:

```
storage:
  trace:
    block:
      encoding: zstd
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

It is important to note that although all of these compression formats are supported in Tempo, at Grafana
we use  zstd and it's possible/probable that the other compression algorithms may have issue at scale.  Please 
file an issue if you stumble upon any problems!