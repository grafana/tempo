// Shared fixture for the dashboard-sharding tests.
//
// Dashboards are just strings of a chosen byte length (std.length(content)
// then equals the size), so scenarios are expressed as plain lists of sizes -
// no real dashboard JSON needed to exercise the sharding maths.
local configmaps = import '../configmaps.libsonnet';

{
  mib:: 1024 * 1024,  // 1048576 - the hard Kubernetes ConfigMap limit
  local kib = 1024,

  // A dashboard "body" of exactly n bytes. Built by string doubling
  // (O(log n) concatenations) - std.repeat/std.join over an n-element array
  // is pathologically slow for the multi-hundred-KB scenarios here.
  pad(n)::
    local grow(s) = if std.length(s) >= n then std.substr(s, 0, n) else grow(s + s);
    if n <= 0 then '' else grow('x'),

  // Scenarios: name -> [ dashboard sizes in bytes ].
  scenarios:: {
    'mimir (real, post-overlay)': [
      534716,
      329748,
      311980,
      172653,
      145276,
      132956,
      119348,
      114080,
      104675,
      85782,
      63065,
      62338,
      52903,
      50425,
      46958,
      45569,
      44220,
      41917,
      41165,
      39386,
      38137,
      33333,
      33241,
      32091,
      31860,
      28223,
      26757,
      17501,
      7249,
      4868,
      4240,
    ],
    // Uniform: balanced's count-split is at its best here (fair baseline).
    'uniform (40x 90KB)': [90 * kib for _ in std.range(1, 40)],
    // Skewed: a few large dashboards among many small ones - the shape that
    // makes a count-based split bunch big dashboards into one shard.
    'skewed (4x 480KB + 30x 20KB)':
      [480 * kib for _ in std.range(1, 4)] + [20 * kib for _ in std.range(1, 30)],
    // Long tail: one dominant dashboard plus a lot of tiny ones.
    'long-tail (1x 700KB + 60x 12KB)':
      [700 * kib] + [12 * kib for _ in std.range(1, 60)],
  },

  // Render a scenario through the REAL library for a given strategy + budget.
  build(sizes, binpack, budget)::
    local grafana = configmaps {
      _config:: {
        configmap_shard_size: budget,
        configmap_binpack: binpack,
        provisioningDir: '/etc/grafana/provisioning',
        labels:: { dashboards: {} },
      },
      grafanaDatasources:: {},
      grafanaNotificationChannels:: {},
      grafanaDashboardFolders:: {
        folder: {
          id: 'folder',
          name: 'folder',
          dashboards: { ['d%03d' % i]: $.pad(sizes[i]) for i in std.range(0, std.length(sizes) - 1) },
        },
      },
    };
    grafana,

  local shardBytes(cms, name) =
    std.foldl(function(a, k) a + std.length(cms[name].data[k]), std.objectFields(cms[name].data), 0),

  // Efficiency + correctness metrics for one (scenario, strategy, budget).
  metrics(sizes, binpack, budget):: {
    local grafana = $.build(sizes, binpack, budget),
    local cms = grafana.dashboard_folders_config_maps,
    local names = std.objectFields(cms),
    local perShard = [shardBytes(cms, n) for n in names],
    local total = std.foldl(function(a, b) a + b, perShard, 0),
    local minShards = std.ceil(total / $.mib),

    // Every dashboard name that ended up in some shard (the library appends
    // a `.json` suffix to each dashboard key).
    local placed = std.set(std.flattenArrays([std.objectFields(cms[n].data) for n in names])),
    local expected = std.set(['d%03d.json' % i for i in std.range(0, std.length(sizes) - 1)]),

    // Shard names must match between the ConfigMaps and the volume mounts,
    // or Grafana would try to mount a shard that does not exist. Dashboard
    // shard mounts live under /grafana/<prefix>/<shard>; take the last path
    // segment to recover <shard>.
    local lastSeg(p) = local parts = std.split(p, '/'); parts[std.length(parts) - 1],
    local mountNames = std.set([
      lastSeg(m.path)
      for m in grafana.configmap_mounts.mounts
      if std.startsWith(m.path, '/grafana/')
    ]),

    shards: std.length(names),
    over_1mib: std.length(std.filter(function(b) b > $.mib, perShard)),
    max_shard: std.foldl(function(a, b) std.max(a, b), perShard, 0),
    total_bytes: total,
    theoretical_min_shards: minShards,
    // shards used vs the theoretical minimum (1.0 == optimal packing)
    shards_vs_min: std.floor(std.length(names) / minShards * 100) / 100,
    // average shard as a % of the 1 MiB budget (higher == denser)
    avg_fill_pct: std.floor(total / std.length(names) / $.mib * 1000) / 10,
    max_pct: std.floor(std.foldl(function(a, b) std.max(a, b), perShard, 0) / $.mib * 1000) / 10,
    // correctness invariants
    all_dashboards_preserved: placed == expected,
    mount_names_match: std.set(names) == mountNames,
  },
}
