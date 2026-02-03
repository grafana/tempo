# Memcached eviction investigation (parquet-page cache)

## What we investigated
- Looked at how Tempo selects memcached servers (not key-choice related to evictions).
- Pulled memcached `stats` and `stats slabs` from a parquet-page memcached pod.
- Checked memory usage (`bytes` vs `limit_maxbytes`) and eviction counters.
- Inspected slab distribution to identify which item sizes were saturated.

## Findings
- **Cache is nearly full**: `bytes` ~1.57 GiB vs `limit_maxbytes` 1.61 GiB (~97.5% used).
- **High churn**: `evictions` ~30k and `evicted_unfetched` ~26.9k indicate many items are evicted before being read.
- **Large-item slab pressure**: slab class **39** (chunk size **524,288 bytes**) dominates usage and is full:
  - `total_pages 1467`, `total_chunks 2934`, `used_chunks 2933`, `free_chunks 1`.
  - This slab class is the busiest (`cmd_set 28908`, `get_hits 31210`).
- **Memory cap reached**: `total_malloced 1610612736` equals the configured `-m 1536` cap.

## Conclusions
- Evictions are **not** due to poor key selection or uneven hashing.
- The parquet-page cache is dominated by ~512KB items, filling the largest slab class and causing evictions.
- The eviction spikes are consistent with **capacity/slab-class pressure**, not distribution issues.

## Suggested actions
- Increase `-m` or add more memcached pods to expand total cache capacity.
- Consider reducing parquet page size if that’s configurable.
- Split caches by size (dedicate a larger cache tier for parquet pages).
- Optionally shorten TTL for large items only, if supported by the application.
