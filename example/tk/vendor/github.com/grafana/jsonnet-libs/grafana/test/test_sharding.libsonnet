// Correctness assertions for dashboard ConfigMap sharding.
// Run via `make tests` (testonnet).
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';
local f = import 'sharding_fixture.libsonnet';

local mimir = f.scenarios['mimir (real, post-overlay)'];
local skewed = f.scenarios['skewed (4x 480KB + 30x 20KB)'];

test.new(std.thisFile)

// --- binpack correctness: never exceeds the ConfigMap limit ---------------
+ test.case.new(
  name='binpack: no shard exceeds budget (mimir @ 900000)',
  test=test.expect.eq(actual=f.metrics(mimir, true, 900000).over_1mib, expected=0),
)
+ test.case.new(
  name='binpack: no shard exceeds budget (skewed @ 900000)',
  test=test.expect.eq(actual=f.metrics(skewed, true, 900000).over_1mib, expected=0),
)

// --- both strategies preserve every dashboard and stay self-consistent ----
+ test.case.new(
  name='binpack: all dashboards preserved (mimir @ 900000)',
  test=test.expect.eq(actual=f.metrics(mimir, true, 900000).all_dashboards_preserved, expected=true),
)
+ test.case.new(
  name='balanced: all dashboards preserved (mimir @ 700000)',
  test=test.expect.eq(actual=f.metrics(mimir, false, 700000).all_dashboards_preserved, expected=true),
)
+ test.case.new(
  name='binpack: mount names match ConfigMap names (mimir @ 900000)',
  test=test.expect.eq(actual=f.metrics(mimir, true, 900000).mount_names_match, expected=true),
)
+ test.case.new(
  name='balanced: mount names match ConfigMap names (mimir @ 700000)',
  test=test.expect.eq(actual=f.metrics(mimir, false, 700000).mount_names_match, expected=true),
)

// --- documents the motivating bug: balanced can overflow the 1 MiB limit --
+ test.case.new(
  name='balanced: overflows 1 MiB on the real post-overlay mimir folder @ 700000',
  test=test.expect.eq(actual=f.metrics(mimir, false, 700000).over_1mib, expected=1),
)
